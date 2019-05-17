// CreatedBy Hu Min
// CreatedAt 2019/5/16 14:40
// Description 本文件下的内容用于向Consul中注册服务
package consulUtil

import (
	"fmt"
	consulApi "github.com/hashicorp/consul/api"
	"log"
	"time"
)

type Register interface {
	Register(info ServiceInfo) error
	Deregister(info ServiceInfo) error
}

//需要注册的服务的信息
type ServiceInfo struct {
	Host        string
	Port        int
	ServiceName string
	//用于更新ttl
	UpdateInterval time.Duration
}

type ConsulRegister struct {
	//consul服务器地址
	ConsulAddress string
	//存活周期
	TTL int
}

var DefaultRegister = ConsulRegister{
	ConsulAddress: "127.0.0.1",
	TTL:           0,
}

//consul配置
var ConsulConfig = consulApi.DefaultConfig()

//获取一个注册器
func NewConsulRegister(ConsulAddress string, TTL int) *ConsulRegister {
	return &ConsulRegister{
		ConsulAddress: ConsulAddress,
		TTL:           TTL,
	}
}

func (c *ConsulRegister) Register(info ServiceInfo) error {
	// initial consul client config
	config := ConsulConfig
	config.Address = c.ConsulAddress
	client, err := consulApi.NewClient(config)
	if err != nil {
		log.Println("新建Consul客户端错误,地址:"+config.Address+":", err.Error())
	}

	serviceId := generateServiceId(info.ServiceName, info.Host, info.Port)

	reg := &consulApi.AgentServiceRegistration{
		ID:      serviceId,
		Name:    info.ServiceName,
		Tags:    []string{info.ServiceName},
		Port:    info.Port,
		Address: info.Host,
	}

	if err = client.Agent().ServiceRegister(reg); err != nil {
		panic(err)
	}

	// 心跳检测
	check := consulApi.AgentServiceCheck{
		TTL:    fmt.Sprintf("%ds", c.TTL),
		Status: consulApi.HealthPassing,
		//超出五倍TTL时间后,从注册中心移除这个服务
		DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.TTL*5),
	}
	err = client.Agent().CheckRegister(
		&consulApi.AgentCheckRegistration{
			ID:                serviceId,
			Name:              info.ServiceName,
			ServiceID:         serviceId,
			AgentServiceCheck: check})
	if err != nil {
		return fmt.Errorf("初始化心跳检测器失败: %s", err.Error())
	}

	//go func() {
	//	ch := make(chan os.Signal, 1)
	//	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	//	x := <-ch
	//	log.Println("LearnGrpc: receive signal: ", x)
	//	// un-register service
	//	cr.DeRegister(info)
	//
	//	s, _ := strconv.Atoi(fmt.Sprintf("%d", x))
	//	os.Exit(s)
	//}()

	//定时发送心跳信息
	go func() {
		ticker := time.NewTicker(info.UpdateInterval)
		for {
			<-ticker.C
			err = client.Agent().UpdateTTL(serviceId, "", check.Status)
			if err != nil {
				log.Println("发送心跳错误: ", err.Error())
			}
		}
	}()

	return nil

}

func (c *ConsulRegister) DeRegister(info ServiceInfo) error {

	serviceId := generateServiceId(info.ServiceName, info.Host, info.Port)

	config := consulApi.DefaultConfig()
	config.Address = c.ConsulAddress
	client, err := consulApi.NewClient(config)
	if err != nil {
		log.Println("新建Consul客户端错误,地址:"+config.Address+":", err.Error())
	}

	err = client.Agent().ServiceDeregister(serviceId)
	if err != nil {
		log.Println("注销服务异常: ", err.Error())
	}
	log.Println("注销服务成功")

	err = client.Agent().CheckDeregister(serviceId)
	if err != nil {
		log.Println("检测当前服务异常: ", err.Error())
	}

	return nil
}

//产生服务的ID
func generateServiceId(name string, host string, port int) string {
	return fmt.Sprintf("%s-%s-%d", name, host, port)
}
