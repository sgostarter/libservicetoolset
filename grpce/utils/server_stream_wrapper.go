package utils

import (
	"context"

	"google.golang.org/grpc"
)

type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func NewServerStreamWrapper(ctx context.Context, s grpc.ServerStream) grpc.ServerStream {
	return &serverStreamWrapper{
		ServerStream: s,
		ctx:          ctx,
	}
}

func (s *serverStreamWrapper) Context() context.Context {
	return s.ctx
}

func (s *serverStreamWrapper) RecvMsg(m interface{}) error {
	return s.ServerStream.RecvMsg(m)
}

func (s *serverStreamWrapper) SendMsg(m interface{}) error {
	return s.ServerStream.SendMsg(m)
}
