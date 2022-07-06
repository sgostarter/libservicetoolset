package interceptors

import (
	"context"
	"time"

	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/fmtutils"
	"github.com/sgostarter/libservicetoolset/grpce/meta"
	"google.golang.org/grpc"
)

func ServerLogInterceptor(logger l.Wrapper) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		id := meta.IdFromOutgoingContext(ctx)

		logger.Infof("[SRV][REQ] id:%v method:%v req:\n%v",
			id, info.FullMethod, fmtutils.Marshal(req))

		st := time.Now()

		res, err := handler(ctx, req)

		logger.Infof("[SRV][RESP] id:%v method:%v cost: %v err:%v data:\n%v;]",
			id, info.FullMethod, time.Since(st), err, fmtutils.Marshal(res))

		return res, err
	}
}

func ServerStreamLogInterceptor(logger l.Wrapper) grpc.StreamServerInterceptor {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		id := meta.IdFromOutgoingContext(ss.Context())

		logger.Infof("[SRV][REQ][STREAM] id:%v method:%v connected", id, info.FullMethod)

		st := time.Now()

		err := handler(srv, ss)

		logger.Infof("[SRV][RESP][STREAM] id:%v method:%v closed. cost:%v", id, info.FullMethod, time.Since(st))

		return err
	}
}

func ClientLogInterceptor(logger l.Wrapper) grpc.UnaryClientInterceptor {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		id := meta.IdFromOutgoingContext(ctx)

		logger.Infof("[CLI][REQ] id:%v method:%v target:%v req:\n%v",
			id, method, cc.Target(), fmtutils.Marshal(req))

		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		logger.Infof("[CLI][RESP] id:%v method:%v cost:%v err:%v data:\n%v",
			id, method, time.Since(start), err, fmtutils.Marshal(reply))

		return err
	}
}

func ClientStreamLogInterceptor(logger l.Wrapper) grpc.StreamClientInterceptor {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer,
		opts ...grpc.CallOption) (stream grpc.ClientStream, err error) {
		id := meta.IdFromOutgoingContext(ctx)

		logger.Infof("[CLI][STREAM] id:%v method:%v connected", id, method)

		return streamer(ctx, desc, cc, method, opts...)
	}
}
