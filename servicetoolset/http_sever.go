package servicetoolset

import (
	"context"
	"net"
	"net/http"

	"github.com/sgostarter/i/l"
)

type HttpServerConfig struct {
	Address           string            `yaml:"address" json:"address"`
	Handler           http.Handler      `json:"-" yaml:"-"`
	DiscoveryExConfig DiscoveryExConfig `yaml:"discovery_ex_config" json:"discovery_ex_config"`
}

type HTTPServer interface {
	Run(ctx context.Context) (err error)
}

func NewHTTPServer(address string, logger l.Wrapper, handler http.Handler) HTTPServer {
	if logger != nil {
		logger = l.NewNopLoggerWrapper()
	}

	return &httpServerImpl{
		address: address,
		logger:  logger.WithFields(l.StringField(l.ClsKey, "httpServerImpl")),
		handler: handler,
	}
}

type httpServerImpl struct {
	address string
	logger  l.Wrapper
	handler http.Handler
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
