package clienttoolset

import (
	"context"
	"fmt"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/librediscovery/discovery"
	"github.com/sgostarter/libservicetoolset/grpce"
	"github.com/sgostarter/libservicetoolset/grpce/interceptors"
	"github.com/sgostarter/libservicetoolset/servicetoolset"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	rrGRpcServerConfig = `
{
	"loadBalancingConfig": [ { "round_robin": {} } ]
}
`
)

type GRPCClientConfig struct {
	Target        string                              `yaml:"target" json:"target"`
	TLSConfig     *servicetoolset.GRPCClientTLSConfig `yaml:"tls_config" json:"tls_config"`
	MetaTransKeys []string                            `json:"-" yaml:"-" ignored:"true"`

	KeepAliveTime    time.Duration `json:"keep_alive_time" yaml:"keep_alive_time"`
	KeepAliveTimeout time.Duration `json:"keep_alive_timeout" yaml:"keep_alive_timeout"`
}

type RegisterSchemasConfig struct {
	Getter  discovery.Getter `json:"-" yaml:"-" ignored:"true"`
	Schemas []string         `yaml:"schemas" json:"schemas"`
}

func DialGRpcServerByName(schema, serverName string, cfg *GRPCClientConfig, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithDefaultServiceConfig(rrGRpcServerConfig))

	if cfg == nil {
		cfg = &GRPCClientConfig{}
	}

	cfg.Target = fmt.Sprintf("%s:///%s", schema, serverName)

	return DialGRPC(cfg, opts)
}

func DialGRPC(cfg *GRPCClientConfig, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	return DialGRPCEx(context.Background(), cfg, opts)
}

func DialGRPCEx(_ context.Context, cfg *GRPCClientConfig, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	dialOptions := make([]grpc.DialOption, 0, len(opts)+1)

	unaryInterceptors := []grpc.UnaryClientInterceptor{
		interceptors.ClientIDInterceptor(cfg.MetaTransKeys),
	}
	streamInterceptors := []grpc.StreamClientInterceptor{
		interceptors.ClientStreamIDInterceptor(cfg.MetaTransKeys),
	}

	dialOptions = append(dialOptions, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(unaryInterceptors...)))
	dialOptions = append(dialOptions, grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(streamInterceptors...)))

	dialOptions = append(dialOptions, opts...)

	if cfg.TLSConfig != nil {
		tlsConfig, err := servicetoolset.GenClientTLSConfig(cfg.TLSConfig)
		if err != nil {
			return nil, err
		}

		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.KeepAliveTime > 0 {
		if cfg.KeepAliveTimeout < time.Second {
			cfg.KeepAliveTimeout = time.Second
		}

		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.KeepAliveTime,    // send pings every x seconds if there is no activity
			Timeout:             cfg.KeepAliveTimeout, // wait x second for ping ack before considering the connection dead
			PermitWithoutStream: true,                 // send pings even without active streams
		}))
	}

	return grpc.NewClient(cfg.Target, dialOptions...)
}

func RegisterSchemas(_ context.Context, cfg *RegisterSchemasConfig, logger l.Wrapper) error {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	if cfg == nil {
		return commerr.ErrInvalidArgument
	}

	if cfg.Getter == nil {
		return commerr.ErrInvalidArgument
	}

	for _, schema := range cfg.Schemas {
		err := grpce.RegisterResolver(cfg.Getter, logger, schema)
		if err != nil {
			logger.Errorf("register schema %v failed: %v", schema, err)
		}
	}

	return nil
}
