package interceptors

import (
	"context"

	"github.com/sgostarter/libservicetoolset/grpce/meta"
	"github.com/sgostarter/libservicetoolset/grpce/utils"
	"google.golang.org/grpc"
)

func ServerIDInterceptor(transKeys []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
		resp interface{}, err error) {
		return handler(meta.TransferContextMeta(ctx, transKeys), req)
	}
}

func ServerStreamIDInterceptor(transKeys []string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := utils.NewServerStreamWrapper(meta.TransferContextMeta(ss.Context(), transKeys), ss)

		return handler(srv, wrapper)
	}
}

func ClientIDInterceptor(transKeys []string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(meta.TransferContextMeta(ctx, transKeys), method, req, reply, cc, opts...)
	}
}

func ClientStreamIDInterceptor(transKeys []string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
		streamer grpc.Streamer, opts ...grpc.CallOption) (stream grpc.ClientStream, err error) {
		return streamer(meta.TransferContextMeta(ctx, transKeys), desc, cc, method, opts...)
	}
}
