package clienttoolset

import (
	"context"
	"fmt"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/commerr"
	"github.com/sgostarter/librediscovery/discovery"
	"github.com/sgostarter/libservicetoolset/grpce"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/sgostarter/libservicetoolset/grpce/interceptors"
	"github.com/sgostarter/libservicetoolset/servicetoolset"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	rrGRpcServerConfig = `
{
	"loadBalancingConfig": [ { "round_robin": {} } ]
}
`
)

type GRPCClientConfig struct {
	Target        string                        `yaml:"target" json:"target"`
	TLSConfig     *servicetoolset.GRPCTlsConfig `yaml:"tls_config" json:"tls_config"`
	MetaTransKeys []string                      `json:"-" yaml:"-" ignored:"true"`
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

	return grpc.Dial(cfg.Target, dialOptions...)
}

func RegisterSchemas(ctx context.Context, cfg *RegisterSchemasConfig, logger l.Wrapper) error {
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
