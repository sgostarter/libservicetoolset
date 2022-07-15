package servicetoolset

import (
	"context"
	"net"
	"net/http"

	"github.com/sgostarter/i/l"
	"github.com/sgostarter/librediscovery/discovery"
)

type HTTPServerConfig struct {
	Name              string            `yaml:"name" json:"name"`
	Address           string            `yaml:"address" json:"address"`
	Handler           http.Handler      `json:"-" yaml:"-"`
	DiscoveryExConfig DiscoveryExConfig `yaml:"discovery_ex_config" json:"discovery_ex_config"`
}

type HTTPServer interface {
	Run(ctx context.Context) (err error)
}

func NewHTTPServer(name, address string, handler http.Handler, discoveryExConfig *DiscoveryExConfig, logger l.Wrapper) HTTPServer {
	if logger != nil {
		logger = l.NewNopLoggerWrapper()
	}

	return &httpServerImpl{
		name:              name,
		address:           address,
		handler:           handler,
		discoveryExConfig: discoveryExConfig,
		logger:            logger.WithFields(l.StringField(l.ClsKey, "httpServerImpl")),
	}
}

type httpServerImpl struct {
	name              string
	address           string
	handler           http.Handler
	discoveryExConfig *DiscoveryExConfig
	logger            l.Wrapper
}

func (impl *httpServerImpl) Run(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	server := &http.Server{
		Handler: impl.handler,
	}

	l, err := net.Listen("tcp", impl.address)
	if err != nil {
		return
	}

	impl.logger.Infof("http server listening on %v", impl.address)

	err = impl.startDiscovery()
	if err != nil {
		impl.logger.Errorf("http server discovery failed: %w", err)
	}

	go func() {
		err = server.Serve(l)
		if err != nil {
			impl.logger.Errorf("http server serve error: %v", err)
		}

		cancel()
	}()

	<-ctx.Done()

	impl.logger.Infof("http server shutting down")

	_ = server.Close()

	return
}

func (impl *httpServerImpl) startDiscovery() error {
	if impl.name == "" || impl.discoveryExConfig == nil || impl.discoveryExConfig.Setter == nil {
		return nil
	}

	host, port, err := GetDiscoveryHostAndPort(impl.discoveryExConfig.ExternalAddress, impl.address)
	if err != nil {
		return err
	}

	serviceInfos := []*discovery.ServiceInfo{
		{
			Host:        host,
			Port:        port,
			ServiceName: discovery.BuildDiscoveryServerName(discovery.TypeBuildInHTTP, impl.name, ""),
		},
	}

	return impl.discoveryExConfig.Setter.Start(serviceInfos)
}
