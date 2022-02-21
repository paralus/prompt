package debug

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"os"
	"os/exec"

	"github.com/RafaySystems/rafay-common/pkg/audit"
	am "github.com/RafaySystems/rafay-common/pkg/auth/middleware"
	ctypesv2 "github.com/RafaySystems/rafay-common/pkg/types/v2"
	"github.com/RafaySystems/ztka/components/prompt/pkg/kube"
	"github.com/RafaySystems/ztka/components/prompt/pkg/prompt"
	"github.com/RafaySystems/ztka/components/prompt/pkg/prompt/completer"
	sentryrpcv2 "github.com/RafaySystems/rafay-sentry/proto/rpc/v2"
	"github.com/RafaySystems/rctl/pkg/hashid"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/xid"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"binary"},
}

type debugHandler struct {
	sp      sentryrpcv2.SentryPool
	tmpPath string
	dev     bool
}

type kubeviewHandler struct {
	dh *debugHandler
}

type getlogsHandler struct {
	dh *debugHandler
}

type reqAuth struct {
	AccountID          ctypesv2.RafayID
	PartnerID          int64
	OrganizationID     int64
	ProjectID          string
	IsSSOUser          bool
	Username           string
	Groups             []string
	IgnoreScopeDefault bool
	GlobalScope        bool
}

func (h *debugHandler) getAuth(r *http.Request, ps httprouter.Params) (*reqAuth, error) {
	meta, err := am.RequestMetaFromContext(r.Context())
	if err != nil {
		return nil, err
	}

	auth := &reqAuth{
		AccountID:          meta.AccountID,
		PartnerID:          meta.PartnerID,
		OrganizationID:     meta.OrganizationID,
		IsSSOUser:          meta.IsSSOUser,
		Username:           meta.Username,
		Groups:             meta.Groups,
		IgnoreScopeDefault: meta.IgnoreScopeDefault,
		GlobalScope:        meta.GlobalScope,
	}

	auth.ProjectID = ps.ByName("project_id")

	return auth, nil
}

func (h *debugHandler) getKubeConfig(ctx context.Context, auth *reqAuth, clusterName, nameSpace string, isSystemSession bool) ([]byte, error) {
	var resp *ctypesv2.HttpBody

	nCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	sc, err := h.sp.NewClient(nCtx)
	if err != nil {
		return nil, err
	}
	defer sc.Close()
	_log.Debugw("get kube config", "auth", auth)
	opts := ctypesv2.QueryOptions{
		PartnerID:          auth.PartnerID,
		OrganizationID:     auth.OrganizationID,
		AccountID:          auth.AccountID,
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
		selector = append(selector, fmt.Sprintf("rafay.dev/clusterName=%s", clusterName))
	}

	opts.Selector = strings.Join(selector, ",")

	if isSystemSession {
		resp, err = sc.GetForClusterSystemSession(nCtx, &sentryrpcv2.GetForClusterRequest{
			QueryOptions: opts,
			Namespace:    nameSpace,
		})
	} else {
		resp, err = sc.GetForClusterWebSession(nCtx, &sentryrpcv2.GetForClusterRequest{
			QueryOptions: opts,
			Namespace:    nameSpace,
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

		_, err = exec.Command("/opt/rafay/kubectl", execArgs...).Output()
		if err == nil {
			cmdExec := exec.Command("/opt/rafay/kubectl", execArgs...)
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

	c, err := kube.NewCompleter(kubeConfig)
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
			kube.NewIOExecutor(rw, uint16(rowsUint), uint16(colsUint), args, event),
			c.Complete,
			prompt.OptionParser(prompt.NewIOParser(uint16(rowsUint), uint16(colsUint), rw)),
			prompt.OptionWriter(prompt.NewIOWriter(rw)),
			prompt.OptionTitle("rafay-prompt: interactive kubernetes client"),
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

func (kh *kubeviewHandler) Handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var decodedCmd string
	var execArgs []string

	auth, err := kh.dh.getAuth(r, ps)
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

	_log.Infow("KubeViewHandle", "post router", ps, "nameSpace", nameSpace, "command", command, "decoded", decodedCmd)

	kubeConfig, err := kh.dh.getKubeConfig(r.Context(), auth, clusterName, nameSpace, true)
	if err != nil {
		_log.Infow("unable to get kube config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// kubeviewHandler is used by DashBoard to get realtime data from cluster
	// Will keep a longer cache for better performance.
	// Cache is purged if there is no update in 24Hr
	dPath := "kubectlview-" + clusterName + strconv.Itoa(int(auth.PartnerID)) + strconv.Itoa(int(auth.OrganizationID)) + strconv.Itoa(int(auth.AccountID))

	args, err := kh.dh.setupPromptEnv(dPath, kubeConfig)
	if err != nil {
		_log.Infow("unable to setup prompt env", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, arg := range args {
		if strings.TrimSpace(arg) != "" {
			execArgs = append(execArgs, arg)
		}
	}

	for _, arg := range strings.Split(decodedCmd, " ") {
		if strings.TrimSpace(arg) != "" {
			execArgs = append(execArgs, arg)
		}
	}

	output, err := exec.Command("/opt/rafay/kubectl", execArgs...).Output()
	if err != nil {
		cmdExec := exec.Command("/opt/rafay/kubectl", execArgs...)
		output, err := cmdExec.CombinedOutput()
		errString := ""
		outString := ""
		if err != nil {
			errString = err.Error()
		}
		if len(output) > 0 {
			outString = string(output)
		}
		http.Error(w, errString+" : "+outString, http.StatusInternalServerError)
	} else {
		w.Write(output)
		_log.Infow("KubeViewHandle", "outputSize", len(output))
	}
}

// NewDebugHandler returns debug handler
func NewDebugHandler(sp sentryrpcv2.SentryPool, tmpPath string, dev bool) (httprouter.Handle, httprouter.Handle, httprouter.Handle) {
	dh := &debugHandler{
		sp:      sp,
		tmpPath: tmpPath,
		dev:     dev,
	}

	kh := &kubeviewHandler{
		dh: dh,
	}

	lh := &getlogsHandler{
		dh: dh,
	}

	return dh.Handle, kh.Handle, lh.Handle
}

func (lh *getlogsHandler) Handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var decodedCmd string

	auth, err := lh.dh.getAuth(r, ps)
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
			if !strings.HasPrefix(decodedCmd, "logs") {
				_log.Infow("getlogsHandler", "cannot use any command other than logs")
				http.Error(w, "invalid cargs", http.StatusInternalServerError)
				return
			}
		} else {
			_log.Infow("getlogsHandler", "failure in decode")
			http.Error(w, "invalid cargs", http.StatusInternalServerError)
			return
		}
	} else {
		_log.Infow("getlogsHandler", "cannot use empty commands in this handler")
		http.Error(w, "invalid cargs", http.StatusInternalServerError)
		return
	}

	_log.Infow("getlogsHandler", "post router", ps, "nameSpace", nameSpace, "command", command, "decoded", decodedCmd)

	kubeConfig, err := lh.dh.getKubeConfig(r.Context(), auth, clusterName, nameSpace, true)
	if err != nil {
		_log.Infow("unable to get kube config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dPath := xid.New().String()

	args, err := lh.dh.setupPromptEnv(dPath, kubeConfig)
	if err != nil {
		_log.Infow("unable to setup prompt env", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	c, err := kube.NewCompleter(kubeConfig)
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
		lh.dh.teardownPromptEnv(dPath)
		return nil
	})

	rw := newWSReadWriter(conn)

	go func() {

		p := prompt.New(
			kube.NewIOExecutor(rw, uint16(rowsUint), uint16(colsUint), args, nil),
			c.Complete,
			prompt.OptionParser(prompt.NewIOParser(uint16(rowsUint), uint16(colsUint), rw)),
			prompt.OptionWriter(prompt.NewIOWriter(rw)),
			prompt.OptionTitle("rafay-prompt: interactive kubernetes client"),
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
	partnerHashID, err := hashid.HashFromInt64(auth.PartnerID)
	if err != nil {
		return nil, err
	}

	orgHashID, err := hashid.HashFromInt64(auth.OrganizationID)
	if err != nil {
		return nil, err
	}

	account := audit.EventActorAccount{
		ID:       auth.AccountID.String(),
		Username: auth.Username,
	}

	event := audit.Event{
		Portal:         "ADMIN",
		PartnerID:      partnerHashID,
		OrganizationID: orgHashID,
		ProjectID:      auth.ProjectID,
		Type:           "kubectl.command.detail",
		Detail:         &audit.EventDetail{},
		Actor: &audit.EventActor{
			Type:           "USER",
			PartnerID:      partnerHashID,
			OrganizationID: orgHashID,
			Account:        account,
			Groups:         auth.Groups,
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
