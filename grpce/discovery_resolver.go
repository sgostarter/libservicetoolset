package grpce

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/commerr"
	"github.com/sgostarter/libeasygo/cuserror"
	"github.com/sgostarter/librediscovery/discovery"
	"google.golang.org/grpc/resolver"
)

var (
	_lock     sync.Mutex
	_builders = make(map[string]*discoveryBuilder) // schema => builder

	_validSchemaServers       sync.Map
	_validGRpcClassToDialName sync.Map
)

func key4CheckServerDiscovery(schema, serverName string) string {
	return schema + ":" + serverName
}

func HasDiscovery(schema, serverName string) bool {
	_, ok := _validSchemaServers.Load(key4CheckServerDiscovery(schema, serverName))

	return ok
}

func GetDialAddressByGRpcClassName(clsName string) string {
	if v, ok := _validGRpcClassToDialName.Load(clsName); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

//
// discoveryBuilder
//

type discoveryBuilder struct {
	getter discovery.Getter
	logger l.Wrapper
	schema string

	resolversLock sync.RWMutex
	resolvers     map[string]map[*discoveryResolver]interface{} // server name => resolver =>

	serviceInfosLock sync.RWMutex
	serviceInfos     map[string][]resolver.Address
}

func newDiscoveryBuilder(getter discovery.Getter, logger l.Wrapper, schema string) (*discoveryBuilder, error) {
	if getter == nil || schema == "" {
		return nil, commerr.ErrInvalidArgument
	}

	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	builder := &discoveryBuilder{
		getter:       getter,
		logger:       logger.WithFields(l.StringField(l.ClsKey, "discoveryBuilder")),
		schema:       schema,
		resolvers:    make(map[string]map[*discoveryResolver]interface{}),
		serviceInfos: make(map[string][]resolver.Address),
	}

	err := builder.getter.Start(builder.onServiceDiscovery, discovery.TypeOption(discovery.TypeBuildInGRPC))
	if err != nil {
		return nil, err
	}

	return builder, nil
}

//
// resolver.Builder
//

func (builder *discoveryBuilder) Build(target resolver.Target, cc resolver.ClientConn,
	_ resolver.BuildOptions) (resolver.Resolver, error) {
	serverName := strings.TrimPrefix(target.URL.Path, "/")
	//serverName := target.Endpoint

	_ = cc.UpdateState(resolver.State{
		Addresses:     builder.resolve(serverName),
		ServiceConfig: nil,
	})

	r := newDiscoveryResolver(builder, serverName, cc)

	builder.resolversLock.Lock()

	defer builder.resolversLock.Unlock()

	if _, ok := builder.resolvers[serverName]; !ok {
		builder.resolvers[serverName] = make(map[*discoveryResolver]interface{})
	}

	builder.resolvers[serverName][r] = time.Now()

	return r, nil
}

func (builder *discoveryBuilder) Scheme() string {
	return builder.schema
}

// discovery callback
func (builder *discoveryBuilder) onServiceDiscovery(services []*discovery.ServiceInfo) {
	serviceInfos := make(map[string][]resolver.Address)

	for _, service := range services {
		_, n, _, err := discovery.ParseDiscoveryServerName(service.ServiceName)
		if err != nil {
			builder.logger.Errorf("parse server name %v failed: %v", service.ServiceName, err)

			continue
		}

		serviceInfos[n] = append(serviceInfos[n], resolver.Address{
			Addr: fmt.Sprintf("%v:%v", service.Host, service.Port),
		})

		_validSchemaServers.Store(key4CheckServerDiscovery(builder.schema, n), time.Now())

		for k, v := range service.Meta {
			if k != discovery.MetaGRPCClass {
				continue
			}

			for _, cls := range strings.Split(v, ";") {
				_validGRpcClassToDialName.Store(cls, fmt.Sprintf("%s:///%s", builder.schema, n))
			}
		}
	}

	builder.serviceInfosLock.Lock()

	defer builder.serviceInfosLock.Unlock()

	builder.serviceInfos = serviceInfos

	go func() {
		builder.resolversLock.RLock()
		defer builder.resolversLock.RUnlock()

		for _, rs := range builder.resolvers {
			for r := range rs {
				r.refresh()
			}
		}
	}()
}

// server name resolver callback
func (builder *discoveryBuilder) resolve(serverName string) []resolver.Address {
	builder.serviceInfosLock.RLock()
	defer builder.serviceInfosLock.RUnlock()

	if addresses, ok := builder.serviceInfos[serverName]; ok {
		return addresses
	}

	return nil
}

func (builder *discoveryBuilder) resolveClosed(r *discoveryResolver) {
	builder.resolversLock.Lock()
	defer builder.resolversLock.Unlock()

	if resolversOnServerName, ok := builder.resolvers[r.serverName]; ok {
		delete(resolversOnServerName, r)
	}

	if len(builder.resolvers[r.serverName]) == 0 {
		delete(builder.resolvers, r.serverName)
	}
}

// RegisterResolver getter和schema一一对应，不应该多个schema公用一个getter，除非getter支持多次Start操作
func RegisterResolver(getter discovery.Getter, logger l.Wrapper, schema string) error {
	var builder *discoveryBuilder

	var err error

	_lock.Lock()

	if _, ok := _builders[schema]; !ok {
		builder, err = newDiscoveryBuilder(getter, logger, schema)
		if err == nil {
			_builders[schema] = builder
		}
	}
	_lock.Unlock()

	if err != nil {
		return err
	}

	if builder == nil {
		return cuserror.NewWithErrorMsg(fmt.Sprintf("schema %v has registered", schema))
	}

	resolver.Register(builder)

	return nil
}

type discoveryResolver struct {
	builder    *discoveryBuilder
	serverName string
	clientConn resolver.ClientConn
}

func newDiscoveryResolver(builder *discoveryBuilder, serverName string, clientConn resolver.ClientConn) *discoveryResolver {
	return &discoveryResolver{
		builder:    builder,
		serverName: serverName,
		clientConn: clientConn,
	}
}

func (r *discoveryResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	r.refresh()
}

func (r *discoveryResolver) Close() {
	r.builder.resolveClosed(r)
}

func (r *discoveryResolver) refresh() {
	_ = r.clientConn.UpdateState(resolver.State{
		Addresses: r.builder.resolve(r.serverName),
	})
}
