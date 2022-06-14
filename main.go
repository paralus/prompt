package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/paralus/paralus/pkg/audit"
	authv3 "github.com/paralus/paralus/pkg/auth/v3"
	logv2 "github.com/paralus/paralus/pkg/log"
	sentryrpcv2 "github.com/paralus/paralus/proto/rpc/sentry"
	systemrpc "github.com/paralus/paralus/proto/rpc/system"
	userrpc "github.com/paralus/paralus/proto/rpc/user"
	"github.com/paralus/prompt/debug"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	apiPortEnv    = "API_PORT"
	sentryAddrEnv = "SENTRY_ADDR"
	tmpPathEnv    = "TEMP_PATH"
	devEnv        = "DEV"
	kubectlBinEnv = "KUBECTL_BIN"
	auditFileEnv  = "AUDIT_LOG_FILE"
	usernameEnv   = "USER_NAME"
)

var (
	apiPort    int
	sentryAddr string
	tmpPath    string
	dev        bool
	kubectlBin string
	auditFile  string

	sp  sentryrpcv2.SentryPool
	pp  systemrpc.SystemPool
	ugp userrpc.UGPool

	_log = logv2.GetLogger()
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"binary"},
}

func setup() {
	viper.SetDefault(apiPortEnv, 7009)
	viper.SetDefault(sentryAddrEnv, "localhost:10000")
	viper.SetDefault(tmpPathEnv, "/tmp")
	viper.SetDefault(devEnv, true)
	viper.SetDefault(kubectlBinEnv, "/usr/local/bin/kubectl")
	viper.SetDefault(auditFileEnv, "/var/log/ztka-prompt/audit.log")
	viper.SetDefault(usernameEnv, "")

	viper.BindEnv(apiPortEnv)
	viper.BindEnv(sentryAddrEnv)
	viper.BindEnv(tmpPathEnv)
	viper.BindEnv(devEnv)
	viper.BindEnv(kubectlBinEnv)
	viper.BindEnv(auditFileEnv)
	viper.BindEnv(usernameEnv)

	apiPort = viper.GetInt(apiPortEnv)
	sentryAddr = viper.GetString(sentryAddrEnv)
	tmpPath = viper.GetString(tmpPathEnv)
	dev = viper.GetBool(devEnv)
	kubectlBin = viper.GetString(kubectlBinEnv)
	auditFile = viper.GetString(auditFileEnv)

	sp = sentryrpcv2.NewSentryPool(sentryAddr, 10)
	pp = systemrpc.NewSystemPool(sentryAddr, 10)
	ugp = userrpc.NewUGPool(sentryAddr, 10)
}

func runAPI(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ao := audit.AuditOptions{
		LogPath:    auditFile,
		MaxSizeMB:  1,
		MaxBackups: 10, // Should we let sidecar do rotation?
		MaxAgeDays: 10, // Make these configurable via env
	}
	auditLogger := audit.GetAuditLogger(&ao)

	dh := debug.NewDebugHandler(sp, pp, ugp, tmpPath, kubectlBin, auditLogger)

	r := httprouter.New()
	r.Handle("GET", "/v2/debug/prompt/project/:project/cluster/:cluster_name", dh)

	n := negroni.New(
		negroni.NewRecovery(),
	)

	if !dev {
		o := authv3.Option{}

		n.Use(authv3.NewRemoteAuthMiddleware(auditLogger, sentryAddr, o))
	}
	n.UseHandler(r)

	s := http.Server{
		Addr:    fmt.Sprintf(":%d", apiPort),
		Handler: n,
	}

	go func() {
		_log.Infow("starting debug prompt server", "port", apiPort)
		err := s.ListenAndServe()
		if err != nil {
			if err == http.ErrServerClosed {
				_log.Infow("debug prompt server shutdown")
				return
			}
			_log.Fatalw("unable to debug prompt server", "error", err)
		}
	}()

	// cleanup unused system sessions cachedir
	pctx, pcancel := context.WithCancel(context.Background())
	defer pcancel()
	go debug.PruneCacheDirs(pctx, tmpPath)

	<-ctx.Done()
	_log.Infow("shutting down debug prompt server")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	s.Shutdown(ctx)
}

func run() {
	ctx := signals.SetupSignalHandler()
	var wg sync.WaitGroup

	wg.Add(1)
	go runAPI(&wg, ctx)

	<-ctx.Done()
	wg.Wait()
}

func main() {
	setup()
	run()
}
