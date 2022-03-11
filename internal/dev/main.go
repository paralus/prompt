package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	logv2 "github.com/RafaySystems/rcloud-base/pkg/log"
	sentryrpcv2 "github.com/RafaySystems/rcloud-base/proto/rpc/sentry"
	"github.com/RafaySystems/prompt/debug"
	"github.com/RafaySystems/prompt/internal/dev/mock"
	ui "github.com/RafaySystems/prompt/internal/dev/ui"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	apiPortEnv    = "API_PORT"
	tmpPathEnv    = "TEMP_PATH"
	kubectlBinEnv = "KUBECTL_BIN"
	auditFileEnv  = "AUDIT_LOG_FILE"
)

var (
	apiPort    int
	tmpPath    string
	kubectlBin string
	auditFile  string
	sp         sentryrpcv2.SentryPool
	_log       = logv2.GetLogger()
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"binary"},
}

func setup() {
	viper.SetDefault(apiPortEnv, 7009)
	viper.SetDefault(tmpPathEnv, "/tmp")
	viper.SetDefault(kubectlBinEnv, "/usr/local/bin/kubectl")
	viper.SetDefault(auditFileEnv, "/var/log/ztka-prompt/audit.log")

	viper.BindEnv(apiPortEnv)
	viper.BindEnv(tmpPathEnv)
	viper.BindEnv(kubectlBinEnv)
	viper.BindEnv(auditFileEnv)

	apiPort = viper.GetInt(apiPortEnv)
	tmpPath = viper.GetString(tmpPathEnv)
	kubectlBin = viper.GetString(kubectlBinEnv)
	auditFile = viper.GetString(auditFileEnv)

	sp = &mock.SentryPool{}
}

func runAPI(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r := httprouter.New()

	dh := debug.NewDebugHandler(sp, tmpPath, kubectlBin, auditFile)

	r.ServeFiles("/v2/debug/ui/*filepath", http.FS(ui.Files))
	r.Handle("GET", "/v2/debug/prompt/project/:project_id/cluster/:cluster_name", dh)

	n := negroni.New(
		negroni.NewRecovery(),
	)
	n.Use(mock.NewDummyAuthMiddleware())
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