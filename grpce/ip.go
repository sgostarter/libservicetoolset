package grpce

import (
	"context"
	"net"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/sgostarter/libeasygo/iputils"
	"google.golang.org/grpc/peer"
)

func GrpcGetRealIP(ctx context.Context) string {
	clientIP := metautils.ExtractIncoming(ctx).Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])

	if clientIP == "" {
		clientIP = strings.TrimSpace(metautils.ExtractIncoming(ctx).Get("X-Real-Ip"))
	}

	if clientIP == "" {
		client, ok := peer.FromContext(ctx)
		if ok && client.Addr != net.Addr(nil) {
			addSlice := strings.Split(client.Addr.String(), ":")
			if addSlice[0] == "[" {
				return "127.0.0.1"
			}

			clientIP = addSlice[0]
		}
	}

	return iputils.RegularIPV4(clientIP)
}
