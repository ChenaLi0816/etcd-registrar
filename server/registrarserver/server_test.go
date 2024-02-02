package registrarserver

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestTxn(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	key := "key"

	s := &EtcdRegistrarServer{etcdModel: &etcdModel{Cli: cli}}
	old, err := s.putGetTxn(ctx, key, 20, func(oldValue string, funcArgs ...string) (string, bool) {
		return oldValue + "1", true
	})
	fmt.Println("old:", old)
}
