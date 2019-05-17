// CreatedBy Hu Min
// CreatedAt 2019/5/16 9:29
// Description
package service

import (
	"context"
	"errors"
	"grpc_cluster/userServuce_1/proto/go"
	"log"
)

type UserService struct {
}

var userMap = map[int32]proto.User{
	1: {
		Id:   1,
		Name: "lisi-service-3",
		Age:  24,
	},
	2: {
		Id:   2,
		Name: "wangwu-service-3",
		Age:  22,
	},
}

func (UserService) GetUserById(ctx context.Context, id *proto.Id) (*proto.User, error) {
	user, ok := userMap[id.Id]
	if ok {
		log.Println(user)
		return &user, nil
	}
	return nil, errors.New("无此id")
}
