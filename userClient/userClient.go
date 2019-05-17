// CreatedBy Hu Min
// CreatedAt 2019/5/16 10:18
// Description
package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"grpc_cluster/consulUtil"
	"grpc_cluster/userServuce_1/proto/go"
	"log"
	"math/rand"
	"time"
)

const (
	serviceName   = "UserService"
	ConsulAddress = "127.0.0.1:8500"
)

func main() {
	schema, err := consulUtil.GenerateAndRegisterConsulResolver(ConsulAddress, serviceName)
	if err != nil {
		log.Fatalln("初始化名称解析器失败", err)
	}
	host := fmt.Sprintf("%s:///%s", schema, serviceName)
	conn, err := grpc.Dial(host, grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
	if err != nil {
		log.Fatalf("连接rpc服务失败: %v", err)
	}
	defer conn.Close()
	client := proto.NewUserServiceClient(conn)

	for {
		time.Sleep(time.Second * 3)

		id := rand.Intn(3)
		timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)
		user, err := client.GetUserById(timeout, &proto.Id{Id: int32(id)})
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(user)
		cancelFunc()
	}

}
