package servicetoolset

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/commerr"
	"github.com/sgostarter/libeasygo/routineman"
	"github.com/sgostarter/librediscovery/discovery"
	"github.com/sgostarter/libservicetoolset/grpce/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type DiscoveryExConfig struct {
	Setter          discovery.Setter  `json:"-" yaml:"-" ignored:"true"`
	ExternalAddress string            `yaml:"external_address" json:"external_address"`
	Meta            map[string]string `yaml:"meta" json:"meta"`
}

type GRPCServerConfig struct {
	Address    string         `yaml:"address" json:"address"`
	TLSConfig  *GRPCTlsConfig `yaml:"tls_config" json:"tls_config"`
	WebAddress string         `yaml:"web_address" json:"web_address"`

	Name              string             `yaml:"name" json:"name"`
	MetaTransKeys     []string           `yaml:"meta_trans_keys" json:"meta_trans_keys"`
	DiscoveryExConfig *DiscoveryExConfig `yaml:"discovery_ex_config" json:"discovery_ex_config"`

	KeepAliveDuration time.Duration `yaml:"keep_alive_duration" json:"keep_alive_duration"`
}

type BeforeServerStart func(server *grpc.Server) error
type GRPCServer interface {
	Start(init BeforeServerStart) (err error)
	Wait()
	Stop()
	StopAndWait()

	Run(ctx context.Context) (err error)
}

func NewGRPCServer(routineMan routineman.RoutineMan, cfg *GRPCServerConfig, opts []grpc.ServerOption, defInit BeforeServerStart,
	logger l.Wrapper, extraInterceptors ...interface{}) (GRPCServer, error) {
	if routineMan == nil {
		routineMan = routineman.NewRoutineMan(context.Background(), logger)
	}

	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	serverOptions := make([]grpc.ServerOption, 0, len(opts)+1)
	serverOptions = append(serverOptions, opts...)

	if cfg.TLSConfig != nil && len(cfg.TLSConfig.Key) > 0 {
		tlsConfig, err := GenServerTLSConfig(cfg.TLSConfig)
		if err != nil {
			return nil, err
		}

		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	impl := &gRPCServerImpl{
		routineMan:        routineMan,
		address:           cfg.Address,
		webAddress:        cfg.WebAddress,
		serverName:        cfg.Name,
		metaTransKeys:     cfg.MetaTransKeys,
		keepaliveDuration: cfg.KeepAliveDuration,
		extraInterceptors: extraInterceptors,
		defInit:           defInit,
		logger:            logger.WithFields(l.StringField(l.ClsKey, "gRPCServerImpl")),
		serverOptions:     serverOptions,
	}

	if cfg.DiscoveryExConfig != nil && cfg.DiscoveryExConfig.Setter != nil {
		impl.setter = cfg.DiscoveryExConfig.Setter
		impl.externalAddress = cfg.DiscoveryExConfig.ExternalAddress
		impl.meta = cfg.DiscoveryExConfig.Meta
	}

	return impl, nil
}

type gRPCServerImpl struct {
	lock sync.Mutex

	routineMan        routineman.RoutineMan
	address           string
	webAddress        string
	serverName        string
	metaTransKeys     []string
	extraInterceptors []interface{}
	keepaliveDuration time.Duration
	serverOptions     []grpc.ServerOption
	defInit           BeforeServerStart
	logger            l.Wrapper

	gRPCListen    net.Listener
	gRPCWebListen net.Listener
	s             *grpc.Server

	setter          discovery.Setter
	externalAddress string
	meta            map[string]string
}

func (impl *gRPCServerImpl) Run(ctx context.Context) (err error) {
	err = impl.Start(nil)
	if err != nil {
		return
	}

	<-ctx.Done()

	impl.StopAndWait()

	return
}

func (impl *gRPCServerImpl) Start(init BeforeServerStart) (err error) {
	impl.lock.Lock()
	defer impl.lock.Unlock()

	if impl.gRPCListen != nil || impl.s != nil {
		err = commerr.ErrAlreadyExists

		return
	}

	if init == nil {
		init = impl.defInit
	}

	if init == nil {
		err = commerr.ErrInvalidArgument

		return
	}

	fnCleanOnFailed := func() {
		if impl.gRPCListen != nil {
			_ = impl.gRPCListen.Close()
			impl.gRPCListen = nil
		}

		if impl.gRPCWebListen != nil {
			_ = impl.gRPCWebListen.Close()
			impl.gRPCWebListen = nil
		}
	}

	impl.gRPCListen, err = net.Listen("tcp", impl.address)
	if err != nil {
		impl.logger.WithFields(l.StringField("gRPCListen", impl.address), l.ErrorField(err)).Error("listenFailed")

		return
	}

	if impl.webAddress != "" {
		impl.gRPCWebListen, err = net.Listen("tcp", impl.webAddress)
		if err != nil {
			impl.logger.WithFields(l.ErrorField(err)).Error("listen4WebFailed")
			fnCleanOnFailed()

			return
		}
	}

	impl.s = grpc.NewServer(impl.getServerOptions()...)

	err = init(impl.s)
	if err != nil {
		impl.logger.WithFields(l.StringField("gRPCListen", impl.address), l.ErrorField(err)).Error("initFailed")
		fnCleanOnFailed()

		return
	}

	reflection.Register(impl.s)

	err = impl.startDiscovery(impl.s)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err)).Error("startDiscovery")
		fnCleanOnFailed()

		return
	}

	impl.routineMan.StartRoutine(impl.mainRoutine, "mainRoutine")

	if impl.gRPCWebListen != nil {
		impl.routineMan.StartRoutine(impl.webRoutine, "webRoutine")
	}

	return
}

func (impl *gRPCServerImpl) webRoutine(ctx context.Context, exiting func() bool) {
	h, err := NewGRPCWebHandler(GRPCWebHandlerInputParameters{
		GRPCServer:          impl.s,
		GRPCWebUseWebsocket: false,
		GRPCWebPingInterval: 0,
	})
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err)).Fatal("NewGRPCWebHandler")
	}

	impl.logger.Info("grpc web server gRPCListen on:", impl.gRPCWebListen.Addr())

	httpServer := &http.Server{Handler: h}
	err = httpServer.Serve(impl.gRPCWebListen)

	if err != nil {
		impl.logger.WithFields(l.ErrorField(err)).Fatal("webServe")
	}
}

func (impl *gRPCServerImpl) mainRoutine(ctx context.Context, exiting func() bool) {
	err := impl.s.Serve(impl.gRPCListen)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err)).Error("GRPCServe")
	}
}

func (impl *gRPCServerImpl) Wait() {
	impl.routineMan.Wait()
}

func (impl *gRPCServerImpl) Stop() {
	if impl.gRPCListen != nil {
		_ = impl.gRPCListen.Close()
	}

	if impl.gRPCWebListen != nil {
		_ = impl.gRPCWebListen.Close()
	}

	impl.routineMan.TriggerStop()
	// impl.s.GracefulStop()
	impl.s.Stop()
}

func (impl *gRPCServerImpl) StopAndWait() {
	impl.Stop()
	impl.routineMan.StopAndWait()
}

func (impl *gRPCServerImpl) getInterceptors() []grpc.ServerOption {
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		interceptors.ServerIDInterceptor(impl.metaTransKeys),
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		interceptors.ServerStreamIDInterceptor(impl.metaTransKeys),
	}

	unaryInterceptors = append(unaryInterceptors, grpc_recovery.UnaryServerInterceptor())
	streamInterceptors = append(streamInterceptors, grpc_recovery.StreamServerInterceptor())

	for _, v := range impl.extraInterceptors {
		// 不是很明白type出来的和直接写func有什么区别，但这俩type在switch的时候确实不一样
		// 而且case用逗号也不行，也很疑惑
		switch interceptor := v.(type) {
		case func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error):
			unaryInterceptors = append(unaryInterceptors, interceptor)
		case func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error:
			streamInterceptors = append(streamInterceptors, interceptor)
		case grpc.UnaryServerInterceptor:
			unaryInterceptors = append(unaryInterceptors, interceptor)
		case grpc.StreamServerInterceptor:
			streamInterceptors = append(streamInterceptors, interceptor)
		default:
			impl.logger.Warn("interceptor not valid")
		}
	}

	return []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
	}
}

func (impl *gRPCServerImpl) getServerOptions() (options []grpc.ServerOption) {
	options = append(options, impl.serverOptions...)
	options = append(options, impl.getInterceptors()...)

	if impl.keepaliveDuration > 0 {
		options = append(options, grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: impl.keepaliveDuration,
		}))
	}

	return
}

func (impl *gRPCServerImpl) startDiscovery(server *grpc.Server) error {
	if impl.setter == nil {
		return nil
	}

	host, port, err := GetDiscoveryHostAndPort(impl.externalAddress, impl.address)
	if err != nil {
		return err
	}

	sis := server.GetServiceInfo()
	classV := ""

	for key := range sis {
		classV += "/" + key + ";"
	}

	if len(classV) > 0 {
		classV = classV[:len(classV)-1]
	}

	meta := map[string]string{discovery.MetaGRPCClass: classV}
	for k, v := range impl.meta {
		meta[k] = v
	}

	serviceInfos := []*discovery.ServiceInfo{
		{
			Host:        host,
			Port:        port,
			ServiceName: discovery.BuildDiscoveryServerName(discovery.TypeBuildInGRPC, impl.serverName, ""),
			Meta:        meta,
		},
	}

	if impl.webAddress != "" {
		host, port, err := GetDiscoveryHostAndPort(impl.externalAddress, impl.webAddress)
		if err != nil {
			impl.logger.Errorf("discovery gRpcWeb on address %v failed: %v", impl.webAddress, err)
		} else {
			serviceInfos = append(serviceInfos, &discovery.ServiceInfo{
				Host:        host,
				Port:        port,
				ServiceName: discovery.BuildDiscoveryServerName(discovery.TypeBuildInGRPCWeb, impl.serverName, ""),
				Meta:        meta,
			})
		}
	}

	return impl.setter.Start(serviceInfos)
}
