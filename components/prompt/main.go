package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/RafaySystems/rafay-common/pkg/auth/interceptors"
	am "github.com/RafaySystems/rafay-common/pkg/auth/middleware"
	authv1 "github.com/RafaySystems/rafay-common/pkg/auth/v1"
	grpcutils "github.com/RafaySystems/rafay-common/pkg/grpc"
	logv2 "github.com/RafaySystems/rafay-common/pkg/log/v2"
	sentryprcv2 "github.com/RafaySystems/rafay-sentry/proto/rpc/v2"
	"github.com/RafaySystems/rcloud-base/components/common/pkg/gateway"
	"github.com/RafaySystems/ztka/components/prompt/debug"
	intdev "github.com/RafaySystems/ztka/components/prompt/internal/dev"
	pbrpcv2 "github.com/RafaySystems/ztka/components/prompt/proto/rpc/v2"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"google.golang.org/grpc"
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

	sp sentryprcv2.SentryPool
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

	sp = sentryprcv2.NewSentryPool(sentryAddr, 10)

	if !dev {
		ap = authv1.NewAuthPool(authAddr, 10)
	}

}

func runAPI(wg *sync.WaitGroup, stop <-chan struct{}) {
	defer wg.Done()
	r := httprouter.New()

	dh := debug.NewDebugHandler(sp, tmpPath, dev)

	r.ServeFiles("/v2/debug/ui/*filepath", intdev.DevFS)
	r.Handle("GET", "/v2/debug/prompt/project/:project_id/cluster/:cluster_name", dh)

	gwHandler, err := gateway.NewGateway(
		context.Background(),
		fmt.Sprintf(":%d", rpcPort),
		make([]runtime.ServeMuxOption, 0),
		pbrpcv2.RegisterDummyHandlerFromEndpoint, // Created a dummy handler otherwise NewGateway throw error on zero handler
	)
	if err != nil {
		_log.Fatalw("unable to create gateway", "error", err)
	}

	r.NotFound = gwHandler

	amOpts := []am.Option{am.WithLogRequest()}
	if dev {
		amOpts = append(amOpts, am.WithDummy())
	} else {
		amOpts = append(amOpts, am.WithAuthPool(ap))
	}

	n := negroni.New(
		negroni.NewRecovery(),
		am.NewAuthMiddleware(amOpts...),
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

	<-stop
	_log.Infow("shutting down debug prompt server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	s.Shutdown(ctx)
}

func runRPC(wg *sync.WaitGroup, stop <-chan struct{}) {
	defer wg.Done()

	var err error

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", rpcPort))
	if err != nil {
		_log.Fatalw("unable to start rpc listener", "error", err)
	}

	var opts []grpc.ServerOption
	if !dev {
		opts = append(opts, grpc.UnaryInterceptor(
			interceptors.NewAuthInterceptor(ap),
		))
	} else {
		opts = append(opts, grpc.UnaryInterceptor(
			interceptors.NewDummyInterceptor()))
	}
	s, err := grpcutils.NewServer(opts...)
	if err != nil {
		_log.Fatalw("unable to create grpc server", "error", err)
	}

	go func() {
		_log.Infow("starting rpc server", "port", rpcPort)
		err = s.Serve(l)
		if err != nil {
			_log.Fatalw("unable to start rpc server", "error", err)
		}
	}()

	<-stop
	s.GracefulStop()
}

func run() {
	stop := signals.SetupSignalHandler()
	var wg sync.WaitGroup

	wg.Add(1)
	go runAPI(&wg, stop)
	go runRPC(&wg, stop)

	<-stop
	wg.Wait()
}

func main() {

	setup()

	run()
}
