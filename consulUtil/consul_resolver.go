// CreatedBy Hu Min
// CreatedAt 2019/5/16 15:15
// Description 本文件下的内容, 用与grpc实现服务注册和发现
package consulUtil

import (
	"context"
	"fmt"
	consulApi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
	"log"
	"sync"
	"time"
)

//实现GRPC下的resolver, 实现服务的名称解析功能,用在客户端
type consulBuilder struct {
	// 目标consul注册中心地址
	address     string
	client      *consulApi.Client
	serviceName string
}

func NewConsulBuilder(address string) resolver.Builder {
	config := ConsulConfig
	config.Address = address
	client, err := consulApi.NewClient(config)
	if err != nil {
		log.Fatal("创建Consul客户端异常", err.Error())
		return nil
	}
	return &consulBuilder{address: address, client: client}
}
func (c *consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	c.serviceName = target.Endpoint

	//获取服务的地址
	adds, serviceConfig, err := c.resolve()
	if err != nil {
		return nil, err
	}
	//adds=nil
	cc.NewAddress(adds)
	cc.NewServiceConfig(serviceConfig)

	consulResolver := newConsulResolver(&cc, c, opts)
	consulResolver.wg.Add(1)
	// 起一个协程 定时更新服务的地址
	go consulResolver.watcher()

	return consulResolver, nil
}

func (consulBuilder) Scheme() string {
	return "consul"
}
func (c *consulBuilder) resolve() ([]resolver.Address, string, error) {

	serviceEntries, _, err := c.client.Health().Service(c.serviceName, "", true, &consulApi.QueryOptions{})
	if err != nil {
		return nil, "", err
	}

	adds := make([]resolver.Address, 0)
	for _, serviceEntry := range serviceEntries {
		address := resolver.Address{Addr: fmt.Sprintf("%s:%d", serviceEntry.Service.Address, serviceEntry.Service.Port)}
		adds = append(adds, address)
	}
	return adds, "", nil
}

type consulResolver struct {
	clientConn           *resolver.ClientConn
	consulBuilder        *consulBuilder
	t                    *time.Ticker
	wg                   sync.WaitGroup
	rn                   chan struct{}
	ctx                  context.Context
	cancel               context.CancelFunc
	disableServiceConfig bool
}

//grpc内部调用,再某个服务无法调用的时候,发送一个信号给watcher方法 更新服务列表
func (c *consulResolver) ResolveNow(resolver.ResolveNowOption) {
	log.Println("ResolveNow")
	select {
	case c.rn <- struct{}{}:
	default:
	}
}

func (c *consulResolver) Close() {
	c.cancel()
	c.wg.Wait()
	c.t.Stop()
}

func newConsulResolver(cc *resolver.ClientConn, cb *consulBuilder, opts resolver.BuildOption) *consulResolver {
	ctx, cancel := context.WithCancel(context.Background())
	return &consulResolver{
		clientConn:           cc,
		consulBuilder:        cb,
		t:                    time.NewTicker(time.Second),
		ctx:                  ctx,
		cancel:               cancel,
		disableServiceConfig: opts.DisableServiceConfig}
}

//用于更新 服务的地址
func (c *consulResolver) watcher() {
	c.wg.Done()
	for {
		select {
		// 如果context被取消。则退出watcher
		case <-c.ctx.Done(): //直接退出更新
			return
		case <-c.t.C: //定时器更新 设置的每秒更新一次
		case <-c.rn: //通道信号更新(没用)
		}
		adds, serviceConfig, err := c.consulBuilder.resolve()
		if err != nil {
			log.Fatal("查询服务异常:", err.Error())
		}
		//adds=nil
		(*c.clientConn).NewAddress(adds)
		(*c.clientConn).NewServiceConfig(serviceConfig)
	}
}

//文档说 不需要实现
type consulClientConn struct {
}

//用于更新 地址
func (cc *consulClientConn) NewAddress(addresses []resolver.Address) {
}
func (cc *consulClientConn) NewServiceConfig(serviceConfig string) {
}
func newConsulClientConn() resolver.ClientConn {
	return &consulClientConn{}
}
func GenerateAndRegisterConsulResolver(address string, serviceName string) (schema string, err error) {
	builder := NewConsulBuilder(address)
	target := resolver.Target{Scheme: builder.Scheme(), Endpoint: serviceName}
	conn := newConsulClientConn()
	_, err = builder.Build(target, conn, resolver.BuildOption{})
	if err != nil {
		return builder.Scheme(), err
	}
	resolver.Register(builder)
	schema = builder.Scheme()
	return
}
