package meta

import (
	"net/url"
	"strings"

	"google.golang.org/grpc/metadata"
)

type MDReaderWriter struct {
	MD metadata.MD
}

func (rw MDReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	rw.MD[key] = append(rw.MD[key], url.QueryEscape(val))
}

func (rw MDReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vs := range rw.MD {
		for _, v := range vs {
			unescapeV, err := url.QueryUnescape(v)
			if err == nil {
				if err := handler(k, unescapeV); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
