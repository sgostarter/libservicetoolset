package utils

import (
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type clientStreamWrapper struct {
	grpc.ClientStream
	desc       *grpc.StreamDesc
	finishFunc func(error)
}

func NewClientStreamWrapper(s grpc.ClientStream, desc *grpc.StreamDesc, finishFunc func(error)) grpc.ClientStream {
	return &clientStreamWrapper{
		ClientStream: s,
		desc:         desc,
		finishFunc:   finishFunc,
	}
}

func (cs *clientStreamWrapper) Header() (metadata.MD, error) {
	md, err := cs.ClientStream.Header()
	if err != nil {
		cs.finishFunc(err)
	}
	return md, err
}

func (cs *clientStreamWrapper) SendMsg(m interface{}) error {
	err := cs.ClientStream.SendMsg(m)
	if err != nil {
		cs.finishFunc(err)
	}
	return err
}

func (cs *clientStreamWrapper) RecvMsg(m interface{}) error {
	err := cs.ClientStream.RecvMsg(m)
	if err == io.EOF {
		cs.finishFunc(nil)
		return err
	}
	if err != nil {
		cs.finishFunc(err)
		return err
	}
	if !cs.desc.ServerStreams {
		cs.finishFunc(nil)
	}
	return err
}

func (cs *clientStreamWrapper) CloseSend() error {
	err := cs.ClientStream.CloseSend()
	if err != nil {
		cs.finishFunc(err)
	}
	return err
}
