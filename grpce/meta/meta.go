package meta

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/sgostarter/libeasygo/cuserror"
	"google.golang.org/grpc/metadata"
)

const (
	// RequestIDOnMetaData unique request id
	RequestIDOnMetaData = "ymi-micro-srv-req-id"
)

func randomNumber() uint64 {
	// nolint: gosec
	return uint64(rand.Int63())
}

func getRandomID() string {
	return fmt.Sprintf("%x", randomNumber())
}

func GetRequestIDFromMD(md metadata.MD) string {
	ids := md.Get(RequestIDOnMetaData)
	if len(ids) > 0 {
		return ids[0]
	}

	return ""
}

/*
func IdFromIncomingContext(ctx context.Context) (id string, newCreateFlag bool) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		id = GetRequestIDFromMD(md)
	}

	if id == "" {
		id = getRandomID()
		newCreateFlag = true
	}
	return
}

func IdToOutgoingContext(ctx context.Context, id string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, RequestIDOnMetaData, id)
}
*/

func IDFromOutgoingContext(ctx context.Context) string {
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		return GetRequestIDFromMD(md)
	}

	return ""
}

// nolint: gocognit
func TransferContextMeta(ctx context.Context, keys []string) context.Context {
	var idInIncomingContext, idInOutgoingContext string

	mdIn, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		mdIn = metadata.New(nil)
	}

	for _, v := range mdIn.Get(RequestIDOnMetaData) {
		if v != "" {
			idInOutgoingContext = v

			break
		}
	}

	mdOut := metadata.New(nil)

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for key, vs := range md {
			if key == RequestIDOnMetaData {
				for _, v := range vs {
					if v != "" {
						idInOutgoingContext = v

						break
					}
				}

				continue
			}

			mdOut.Set(key, vs...)
		}
	}

	if keys == nil {
		keys = make([]string, 0, mdIn.Len())

		for key := range mdIn {
			if key == RequestIDOnMetaData {
				continue
			}

			keys = append(keys, key)
		}
	}

	if idInIncomingContext == "" {
		idInIncomingContext = idInOutgoingContext
	}

	if idInIncomingContext == "" {
		idInIncomingContext = getRandomID()
	}

	for _, key := range keys {
		if key == RequestIDOnMetaData {
			continue
		}

		if len(mdIn[key]) == 0 {
			continue
		}

		if len(mdOut[key]) > 0 {
			continue
		}

		if strings.EqualFold(key, "connection") {
			continue
		}

		mdOut.Set(key, mdIn[key]...)
	}

	mdOut.Set(RequestIDOnMetaData, idInIncomingContext)

	return metadata.NewOutgoingContext(ctx, mdOut)
}

func GetFromMetaString(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get(key)
		if len(values) > 0 {
			return values[0], nil
		}
	}

	return "", cuserror.NewWithErrorMsg("not exists")
}

func GetStringFromMeta(ctx context.Context, key string) (string, error) {
	return GetFromMetaString(ctx, key)
}

func GetIntFromMeta(ctx context.Context, key string) (int64, error) {
	v, err := GetStringFromMeta(ctx, key)
	if err != nil {
		return 0, err
	}

	vv, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}

	return vv, nil
}
