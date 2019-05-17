// CreatedBy Hu Min
// CreatedAt 2019/5/16 9:20
// Description
package main

import (
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"grpc_cluster/consulUtil"
	"grpc_cluster/userServuce_1/proto/go"
	"grpc_cluster/userServuce_1/service"
	"log"
	"net"
	"time"
)

const (
	servicePort = 9101
	serviceHost = "127.0.0.1"
	serviceName = "UserService"
	consulPort  = 8500
	consulHost  = "127.0.0.1"
)

func main() {

	listener, e := net.Listen("tcp", ":9101")
	if e != nil {
		log.Println(e)
	}
	server := grpc.NewServer()
	register := consulUtil.NewConsulRegister(fmt.Sprintf("%s:%d", consulHost, consulPort), 15)
	_ = register.Register(consulUtil.ServiceInfo{
		Host:           serviceHost,
		Port:           servicePort,
		ServiceName:    serviceName,
		UpdateInterval: time.Second,
	})
	proto.RegisterUserServiceServer(server, service.UserService{})

	//如果启动了gprc反射服务，那么就可以通过reflection包提供的反射服务查询gRPC服务或调用gRPC方法。\
	reflection.Register(server)

	log.Println(server.Serve(listener))
}
