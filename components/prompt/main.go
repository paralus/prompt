package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	authv1 "github.com/RafaySystems/rafay-common/pkg/auth/v1"
	logv2 "github.com/RafaySystems/rafay-common/pkg/log/v2"
	authv3 "github.com/RafaySystems/rcloud-base/components/common/pkg/auth/v3"
	sentryrpcv2 "github.com/RafaySystems/rcloud-base/components/common/proto/rpc/sentry"
	"github.com/RafaySystems/ztka/components/prompt/debug"
	intdev "github.com/RafaySystems/ztka/components/prompt/internal/dev"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	authAddrEnv   = "AUTH_ADDR"
	apiPortEnv    = "API_PORT"
	sentryAddrEnv = "SENTRY_ADDR"
	tmpPathEnv    = "TEMP_PATH"
	devEnv        = "DEV"
	rpcPortEnv    = "RPC_PORT"
)

var (
	apiPort    int
	sentryAddr string
	tmpPath    string
	dev        bool
	authAddr   string
	rpcPort    int

	sp sentryrpcv2.SentryPool
	ap authv1.AuthPool

	_log = logv2.GetLogger()
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"binary"},
}

func setup() {
	viper.SetDefault(apiPortEnv, 7009)
	viper.SetDefault(rpcPortEnv, 7010)
	viper.SetDefault(sentryAddrEnv, "localhost:10000")
	viper.SetDefault(tmpPathEnv, "/tmp")
	viper.SetDefault(devEnv, true)
	viper.SetDefault(authAddrEnv, "authsrv.rcloud-admin.svc.cluster.local:50011")

	viper.BindEnv(apiPortEnv)
	viper.BindEnv(rpcPortEnv)
	viper.BindEnv(sentryAddrEnv)
	viper.BindEnv(tmpPathEnv)
	viper.BindEnv(devEnv)
	viper.BindEnv(authAddrEnv)

	apiPort = viper.GetInt(apiPortEnv)
	rpcPort = viper.GetInt(rpcPortEnv)
	sentryAddr = viper.GetString(sentryAddrEnv)
	tmpPath = viper.GetString(tmpPathEnv)
	dev = viper.GetBool(devEnv)
	authAddr = viper.GetString(authAddrEnv)

	sp = sentryrpcv2.NewSentryPool(sentryAddr, 10)

	if !dev {
		ap = authv1.NewAuthPool(authAddr, 10)
	}

}

func runAPI(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r := httprouter.New()

	dh := debug.NewDebugHandler(sp, tmpPath, dev)

	r.ServeFiles("/v2/debug/ui/*filepath", intdev.DevFS)
	r.Handle("GET", "/v2/debug/prompt/project/:project_id/cluster/:cluster_name", dh)

	ac := authv3.NewAuthContext()
	o := authv3.Option{}

	n := negroni.New(
		negroni.NewRecovery(),
		ac.NewAuthMiddleware(o),
	)

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
