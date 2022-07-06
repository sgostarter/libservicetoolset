package servicetoolset

import (
	"net/http"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCWebHandler struct {
	gRPCServer          *grpc.Server
	gRPCWebUseWebsocket bool
	gRPCWebPingInterval time.Duration
}

type GRPCWebHandlerInputParameters struct {
	GRPCServer          *grpc.Server
	GRPCWebUseWebsocket bool
	GRPCWebPingInterval time.Duration
}

func NewGRPCWebHandler(parameters GRPCWebHandlerInputParameters) (http.Handler, error) {
	if parameters.GRPCServer == nil {
		return nil, status.Error(codes.InvalidArgument, "")
	}

	return &gRPCWebHandler{
		gRPCServer:          parameters.GRPCServer,
		gRPCWebUseWebsocket: parameters.GRPCWebUseWebsocket,
		gRPCWebPingInterval: parameters.GRPCWebPingInterval,
	}, nil
}

func (s *gRPCWebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	options := []grpcweb.Option{
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	}
	if s.gRPCWebUseWebsocket {
		options = append(
			options,
			grpcweb.WithWebsockets(true),
			grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool { return true }),
		)

		if s.gRPCWebPingInterval > 0 {
			options = append(options, grpcweb.WithWebsocketPingInterval(s.gRPCWebPingInterval))
		}
	}

	wrappedGrpc := grpcweb.WrapServer(s.gRPCServer, options...)
	wrappedGrpc.ServeHTTP(w, r)
}
