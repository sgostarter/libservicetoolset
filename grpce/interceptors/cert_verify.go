package interceptors

import (
	"context"
	"errors"

	"github.com/sgostarter/libeasygo/certutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func ServerVerifyInterceptor(secureOption *certutils.SecureOption) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if secureOption != nil && secureOption.ServerWithTLS {
			err := VerifyClientCert(ctx, secureOption)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, "verify client cert failed: %v", err)
			}
		}
		return handler(ctx, req)
	}
}

func ServerStreamVerifyInterceptor(secureOption *certutils.SecureOption) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if secureOption != nil && secureOption.ServerWithTLS {
			err := VerifyClientCert(ss.Context(), secureOption)
			if err != nil {
				return status.Errorf(codes.Unauthenticated, "verify client cert failed: %v", err)
			}
		}
		return handler(srv, ss)
	}
}

// VerifyClientCert .
func VerifyClientCert(ctx context.Context, secureOption *certutils.SecureOption) error {
	clientPeer, ok := peer.FromContext(ctx)
	if !ok {
		return errors.New("cert miss")
	}

	tlsInfo, ok := clientPeer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return errors.New("what's wrong")
	}
	if len(tlsInfo.State.PeerCertificates) < 1 {
		return errors.New("cert chain miss")
	}
	for _, v := range tlsInfo.State.PeerCertificates {
		ok, err := certutils.VerifyCertPublicKey(v.PublicKey, v.Subject.CommonName, secureOption)
		if err != nil {
			return err
		}
		if !ok {
			err = errors.New("VerifyCertPublicKey for PeerCertificates, result is false")
			return err
		}
	}
	return nil
}
