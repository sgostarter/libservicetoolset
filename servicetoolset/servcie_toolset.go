package servicetoolset

import (
	"context"
	"strings"

	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/commerr"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
)

type ServerToolset struct {
	ctx          context.Context
	logger       l.Wrapper
	serverHelper *ServerHelper
	gRPCServer   GRPCServer
	httpServer   HTTPServer

	started atomic.Bool
}

func NewServerToolset(ctx context.Context, logger l.Wrapper) *ServerToolset {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	sst := &ServerToolset{
		ctx:    ctx,
		logger: logger.WithFields(l.StringField(l.ClsKey, "ServerToolset")),
	}

	sst.serverHelper = NewServerHelper(ctx, logger)

	return sst
}

func (st *ServerToolset) CreateGRpcServer(cfg *GRPCServerConfig, opts []grpc.ServerOption, beforeServerStart BeforeServerStart) (err error) {
	if st.gRPCServer != nil {
		err = commerr.ErrAlreadyExists

		return
	}

	st.gRPCServer, err = NewGRPCServer(nil, cfg, opts, beforeServerStart, st.logger)
	if err != nil {
		return
	}

	return
}

func (st *ServerToolset) CreateHTTPServer(cfg *HTTPServerConfig) error {
	if st.httpServer != nil {
		return commerr.ErrAlreadyExists
	}

	if cfg == nil || cfg.Address == "" || !strings.Contains(cfg.Address, ":") || cfg.Handler == nil {
		return commerr.ErrInvalidArgument
	}

	st.httpServer = NewHTTPServer(cfg.Name, cfg.Address, cfg.Handler, &cfg.DiscoveryExConfig, st.logger)

	return nil
}

func (st *ServerToolset) Start() error {
	if !st.started.CAS(false, true) {
		return commerr.ErrAlreadyExists
	}

	if st.gRPCServer != nil {
		st.serverHelper.StartServer(st.gRPCServer)
	}

	if st.httpServer != nil {
		st.serverHelper.StartServer(st.httpServer)
	}

	return nil
}

func (st *ServerToolset) Wait() {
	_ = st.Start()
	st.serverHelper.Wait()
}
