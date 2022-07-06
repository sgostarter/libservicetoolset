package servicetoolset

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sgostarter/i/l"
)

func SignalContext(ctx context.Context, logger l.Wrapper) context.Context {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	ctx, cancel := context.WithCancel(ctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("listening for shutdown signal")

		<-sigs
		logger.Info("shutdown signal received")

		signal.Stop(sigs)
		close(sigs)
		cancel()
	}()

	return ctx
}

type AbstractServer interface {
	Run(ctx context.Context) error
}

type ServerHelper struct {
	ctx    context.Context
	wg     sync.WaitGroup
	logger l.Wrapper
}

func NewServerHelper(ctx context.Context, logger l.Wrapper) *ServerHelper {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return &ServerHelper{
		ctx:    SignalContext(ctx, logger),
		logger: logger.WithFields(l.StringField(l.ClsKey, "ServerHelper")),
	}
}

func (sh *ServerHelper) StartServer(s AbstractServer) {
	sh.wg.Add(1)

	go func() {
		defer sh.wg.Done()

		if err := s.Run(sh.ctx); err != nil {
			sh.logger.Fatalf("runServer error:%v", err)
		}
	}()
}

func (sh *ServerHelper) Wait() {
	sh.wg.Wait()
}
