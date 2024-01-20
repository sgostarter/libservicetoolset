package samples

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/stg/redisex"
	"github.com/sgostarter/libeasygo/ut"
	"github.com/sgostarter/librediscovery"
	"github.com/sgostarter/libservicetoolset/clienttoolset"
	"github.com/sgostarter/libservicetoolset/grpce"
	"github.com/sgostarter/libservicetoolset/servicetoolset"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
)

type TestHelloWorld struct {
	helloworld.UnimplementedGreeterServer

	id string
}

func (o *TestHelloWorld) SayHello(_ context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{
		Message: fmt.Sprintf("Hi %v, I'm %v", req.Name, o.id),
	}, nil
}

// nolint
func Test(t *testing.T) {
	serverName := "testsvr"

	cfg := ut.SetupUTConfig4Redis(t)
	redisClient, err := redisex.InitRedis(cfg.RedisDNS)
	assert.Nil(t, err)

	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := l.NewConsoleLoggerWrapper()

	getter, err := librediscovery.NewGetter(ctx, logger, redisClient, "", 10*time.Second, time.Second)
	assert.Nil(t, err)

	go func() {
		setter, err := librediscovery.NewSetter(ctx, logger, redisClient, "", time.Second)
		assert.Nil(t, err)

		serviceToolset := servicetoolset.NewServerToolset(ctx, logger)
		err = serviceToolset.CreateGRpcServer(&servicetoolset.GRPCServerConfig{
			Address:       ":9001",
			TLSConfig:     nil,
			WebAddress:    "",
			Name:          serverName + ":1",
			MetaTransKeys: nil,
			DiscoveryExConfig: &servicetoolset.DiscoveryExConfig{
				Setter:          setter,
				ExternalAddress: "127.0.0.1",
			},
		}, nil, func(server *grpc.Server) error {
			helloworld.RegisterGreeterServer(server, &TestHelloWorld{
				id: "node1",
			})

			return nil
		})

		assert.Nil(t, err)
		err = serviceToolset.Start()
		assert.Nil(t, err)
		serviceToolset.Wait()
	}()

	go func() {
		setter, err := librediscovery.NewSetter(ctx, logger, redisClient, "", time.Second)
		assert.Nil(t, err)

		serviceToolset := servicetoolset.NewServerToolset(ctx, logger)
		err = serviceToolset.CreateGRpcServer(&servicetoolset.GRPCServerConfig{
			Name:          serverName + ":2",
			Address:       ":9002",
			MetaTransKeys: nil,
			DiscoveryExConfig: &servicetoolset.DiscoveryExConfig{
				Setter:          setter,
				ExternalAddress: "127.0.0.1",
			},
		}, nil, func(server *grpc.Server) error {
			helloworld.RegisterGreeterServer(server, &TestHelloWorld{
				id: "node2",
			})

			return nil
		})

		assert.Nil(t, err)
		err = serviceToolset.Start()
		assert.Nil(t, err)
		serviceToolset.Wait()
	}()

	schema := "rediscoverytest"

	err = grpce.RegisterResolver(getter, logger, schema)
	assert.Nil(t, err)

	conn, err := clienttoolset.DialGRPC(&clienttoolset.GRPCClientConfig{
		Target: fmt.Sprintf("%s:///%s", schema, serverName),
	}, []grpc.DialOption{grpc.WithDefaultServiceConfig(`
{
	"loadBalancingConfig": [ { "round_robin": {} } ]
}
`)})
	assert.Nil(t, err)

	cli := helloworld.NewGreeterClient(conn)
	for idx := 0; idx < 10; idx++ {
		time.Sleep(time.Second)

		resp, err := cli.SayHello(context.Background(), &helloworld.HelloRequest{
			Name: "tester1",
		})
		if err != nil {
			t.Logf("error: %v", err)

			continue
		}
		t.Log(resp.Message)
	}
}
