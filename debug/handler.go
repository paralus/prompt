package debug

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"os"
	"os/exec"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/paralus/paralus/pkg/audit"
	"github.com/paralus/paralus/pkg/service"
	sentryrpcv2 "github.com/paralus/paralus/proto/rpc/sentry"
	systemrpc "github.com/paralus/paralus/proto/rpc/system"
	userrpc "github.com/paralus/paralus/proto/rpc/user"
	ctypesv3 "github.com/paralus/paralus/proto/types/commonpb/v3"
	"github.com/paralus/prompt/pkg/kube"
	"github.com/paralus/prompt/pkg/prompt"
	"github.com/paralus/prompt/pkg/prompt/completer"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"binary"},
}

type debugHandler struct {
	sp          sentryrpcv2.SentryPool
	pp          systemrpc.SystemPool
	ugp         userrpc.UGPool
	tmpPath     string
	kubectlBin  string
	auditLogger *zap.Logger
}

type reqAuth struct {
	Account            string
	Partner            string
	Organization       string
	ProjectID          string
	Project            string
	IsSSOUser          bool
	Username           string
	Groups             []string
	IgnoreScopeDefault bool
	GlobalScope        bool
}

func (h *debugHandler) getAuth(r *http.Request, ps httprouter.Params) (*reqAuth, error) {

	sd, ok := service.GetSessionDataFromContext(r.Context())
	if !ok {
		return nil, errors.New("failed to get session data")
	}
	_log.Infow("running in production mode", "account", sd.Account)

	auth := &reqAuth{
		Account:      sd.GetAccount(),
		Partner:      sd.GetPartner(),
		Organization: sd.GetOrganization(),
		ProjectID:    sd.GetProject().GetList()[0].GetProjectId(),
		Project:      sd.GetProject().GetList()[0].GetProject(),
		IsSSOUser:    sd.GetIsSsoUser(),
		Username:     sd.GetUsername(),
		Groups:       sd.GetGroups(),
	}

	return auth, nil
}

func (h *debugHandler) getKubeConfig(ctx context.Context, auth *reqAuth, clusterName, nameSpace string, isSystemSession bool) ([]byte, error) {
	var resp *ctypesv3.HttpBody

	nCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	sc, err := h.sp.NewClient(nCtx)
	if err != nil {
		return nil, err
	}
	defer sc.Close()
	_log.Debugw("get kube config", "auth", auth)
	opts := ctypesv3.QueryOptions{
		Partner:            auth.Partner,
		Organization:       auth.Organization,
		Account:            auth.Account,
		IsSSOUser:          auth.IsSSOUser,
		Username:           auth.Username,
		Groups:             auth.Groups,
		IgnoreScopeDefault: auth.IgnoreScopeDefault,
		GlobalScope:        auth.GlobalScope,
	}

	var selector []string

	if auth.ProjectID != "" && auth.ProjectID != "all" {
		selector = append(selector, fmt.Sprintf("project/%s", auth.ProjectID))
	}
	if clusterName != "all" {
		selector = append(selector, fmt.Sprintf("paralus.dev/clusterName=%s", clusterName))
	}

	opts.Selector = strings.Join(selector, ",")

	if isSystemSession {
		resp, err = sc.GetForClusterSystemSession(nCtx, &sentryrpcv2.GetForClusterRequest{
			Opts:      &opts,
			Namespace: nameSpace,
		})
	} else {
		resp, err = sc.GetForClusterWebSession(nCtx, &sentryrpcv2.GetForClusterRequest{
			Opts:      &opts,
			Namespace: nameSpace,
		})
	}

	if err != nil {
		_log.Infow("unable to get kubeconfig using GetForClusterWebSession", "error", err)
		return nil, err
	}
	return resp.Data, nil

}

func (h *debugHandler) setupPromptEnv(dPath string, kubeConfig []byte) (args []string, err error) {
	path := fmt.Sprintf("%s/%s", h.tmpPath, dPath)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return
	}

	kubeConfigPath := fmt.Sprintf("%s/kubeconfig.yaml", path)

	err = ioutil.WriteFile(kubeConfigPath, kubeConfig, 0644)
	if err != nil {
		return
	}

	args = append(args, fmt.Sprintf("--cache-dir=%s", path))
	args = append(args, fmt.Sprintf("--kubeconfig=%s", kubeConfigPath))

	return
}

func (h *debugHandler) teardownPromptEnv(dPath string) {
	os.RemoveAll(fmt.Sprintf("%s/%s", h.tmpPath, dPath))
}

func (h *debugHandler) Handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var decodedCmd string

	auth, err := h.getAuth(r, ps)
	if err != nil {
		_log.Infow("unable to get auth", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	clusterName := ps.ByName("cluster_name")

	nameSpace := r.URL.Query().Get("namespace")

	command := r.URL.Query().Get("cargs")
	if command != "" {
		decod, err := base64.StdEncoding.DecodeString(command)
		if err == nil {
			decodedCmd = string(decod)
		}
	}
	_log.Infow("Handle", "post router", ps, "nameSpace", nameSpace, "command", command, "decoded", decodedCmd)

	kubeConfig, err := h.getKubeConfig(r.Context(), auth, clusterName, nameSpace, false)
	if err != nil {
		_log.Infow("unable to get kube config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dPath := xid.New().String()

	args, err := h.setupPromptEnv(dPath, kubeConfig)
	if err != nil {
		_log.Infow("unable to setup prompt env", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		// prime cache for faster initial response
		var execArgs []string
		for _, arg := range args {
			if strings.TrimSpace(arg) != "" {
				execArgs = append(execArgs, arg)
			}
		}

		execArgs = append(execArgs, "api-resources")

		_, err = exec.Command(h.kubectlBin, execArgs...).Output()
		if err == nil {
			cmdExec := exec.Command(h.kubectlBin, execArgs...)
			err = cmdExec.Run()
		}
	}()

	rows := r.URL.Query().Get("rows")
	cols := r.URL.Query().Get("cols")

	rowsUint, err := strconv.ParseUint(rows, 10, 16)
	if err != nil {
		_log.Infow("unable to parse rows", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	colsUint, err := strconv.ParseUint(cols, 10, 16)
	if err != nil {
		_log.Infow("unable to parse cols", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c, err := kube.NewCompleter(context.Background(), kubeConfig)
	if err != nil {
		_log.Infow("unable to create completer", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_log.Infow("unable to update", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	conn.SetCloseHandler(func(code int, text string) error {
		_log.Infow("client closed websocket")
		cancel()
		h.teardownPromptEnv(dPath)
		return nil
	})

	rw := newWSReadWriter(conn)

	event, err := h.GetEventForKubectlCommands(r, auth, clusterName)
	if err != nil {
		_log.Infow("unable to get audit for kubectl commands", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {

		p := prompt.New(
			kube.NewIOExecutor(rw, uint16(rowsUint), uint16(colsUint), args, event, h.kubectlBin, h.auditLogger),
			c.Complete,
			prompt.OptionParser(prompt.NewIOParser(uint16(rowsUint), uint16(colsUint), rw)),
			prompt.OptionWriter(prompt.NewIOWriter(rw)),
			prompt.OptionTitle("paralus-prompt: interactive kubernetes client"),
			prompt.OptionPrefix("kubectl "),
			prompt.OptionPrefixTextColor(prompt.Green),
			prompt.OptionInputTextColor(prompt.Yellow),
			prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
			prompt.OptionSwitchKeyBindMode(prompt.CommonKeyBind),
		)
		if decodedCmd != "" {
			p.RunPreset(ctx, decodedCmd)
		} else {
			p.Run(ctx)
		}
	}()

	<-ctx.Done()
	_log.Infow("closing websocket context done")

}

// NewDebugHandler returns debug handler
func NewDebugHandler(sp sentryrpcv2.SentryPool, pp systemrpc.SystemPool, ugp userrpc.UGPool, tmpPath, kubectlBin string, auditLogger *zap.Logger) httprouter.Handle {
	dh := &debugHandler{
		sp:          sp,
		pp:          pp,
		ugp:         ugp,
		tmpPath:     tmpPath,
		kubectlBin:  kubectlBin,
		auditLogger: auditLogger,
	}

	return dh.Handle
}

func isDevMode() bool {
	return viper.GetBool("DEV")
}

func getUsername() string {
	return viper.GetString("USER_NAME")
}

func hasKubeCache(path string) bool {
	if strings.Contains(path, "kubectlview-") {
		s := strings.Split(path, "/")
		if len(s) == 3 {
			return true
		}
	}
	return false
}

func isStaleDir(cacheDir string) bool {
	info, err := os.Stat(cacheDir)
	if err != nil {
		_log.Infow("unable to stat file", "cacheDir", cacheDir, "error", err)
		return false
	}

	// delete directory that has no modified files in a day.
	if info.ModTime().Before(time.Now().Add(-time.Hour * 24 * 1)) {
		_log.Debugw("cache dir not updated for a day", "cachedir", cacheDir)
		return true
	}

	return false
}

func staleCacheDirCheck(root string) {
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}

	var staleDir []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() && hasKubeCache(path) && isStaleDir(path) {
			staleDir = append(staleDir, path)
		}
		return nil
	})

	if err != nil {
		_log.Infow("failed in filepath.Walk to find stale cache dir", "error", err)
		return
	}

	for _, staleDir := range staleDir {
		err := os.RemoveAll(staleDir)
		if err != nil {
			_log.Infow("unable to remove stale cache dir", "error", err, "staleDir", staleDir)
		} else {
			_log.Infow("remove stale cache dir", "staleDir", staleDir)
		}
	}
}

func PruneCacheDirs(ctx context.Context, root string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			go staleCacheDirCheck(root)
		}
	}
}

func (h *debugHandler) GetEventForKubectlCommands(r *http.Request, auth *reqAuth, clusterName string) (*audit.Event, error) {
	account := audit.EventActorAccount{
		Username: auth.Username,
	}

	event := audit.Event{
		Portal:  "ADMIN",
		Project: auth.Project,
		Type:    "kubectl.command.detail",
		Detail:  &audit.EventDetail{},
		Actor: &audit.EventActor{
			Type:    "USER",
			Account: account,
			Groups:  auth.Groups,
		},
		Client: &audit.EventClient{
			Type:      "BROWSER",
			IP:        r.Header.Get("X-Forwarded-For"),
			UserAgent: r.UserAgent(),
			Host:      r.Host,
		},
	}
	event.Detail.Meta = make(map[string]string)
	event.Detail.Meta["cluster_name"] = clusterName
	return &event, nil
}
