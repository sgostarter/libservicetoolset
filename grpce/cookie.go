package grpce

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sgostarter/libeasygo/strutils"
	"github.com/sgostarter/libservicetoolset/grpce/meta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// SetHttpCookie .
func SetHttpCookie(ctx context.Context, key, val string, maxAge int) error {
	domain := strutils.StringTrim(domainFromContext(ctx))
	if domain == "" {
		return errors.New("no domain checked")
	}

	cookie := http.Cookie{
		Domain:   domain,
		Name:     key,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   maxAge,
	}
	return grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", cookie.String()))
}

// UnsetHttpCookie .
func UnsetHttpCookie(ctx context.Context, key string) error {
	domain := strutils.StringTrim(domainFromContext(ctx))
	if domain == "" {
		return errors.New("no domain checked")
	}
	cookie := http.Cookie{
		Domain:   domain,
		Name:     key,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().AddDate(-1, 0, 0),
	}
	return grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", cookie.String()))
}

// GetCookieStringFromContext .
func GetCookieStringFromContext(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get("cookie")
	values = append(values, md.Get("ymicookie")...)
	if len(values) == 0 {
		return ""
	}
	for _, value := range values {
		cookies := strings.Split(value, ";")
		for _, cookie := range cookies {
			ps := strings.SplitN(cookie, "=", 2)
			if len(ps) != 2 {
				continue
			}
			if strutils.StringTrim(ps[0]) == key {
				return strutils.StringTrim(ps[1])
			}
		}
	}
	return ""
}

func domainFromContext(ctx context.Context) string {
	domain := ""
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("origin")
		if len(values) > 0 {
			domain = values[0]
		}
	}
	idx := strings.Index(domain, "://")
	if idx != -1 {
		domain = domain[idx+3:]
	}
	idx = strings.Index(domain, ":")
	if idx != -1 {
		domain = domain[0:idx]
	}
	return domain
}

// GetStringFromContext .
func GetStringFromContext(ctx context.Context, key string) string {
	token := GetCookieStringFromContext(ctx, key)
	if token != "" {
		return token
	}
	token, _ = meta.GetStringFromMeta(ctx, key)
	return token
}
