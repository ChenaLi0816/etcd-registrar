package registrarclient

import (
	"context"
	"log"
	"testing"
)

const (
	NAME    = "test"
	ADDRESS = "test_address"
)

//func TestNewRegistrarClient(t *testing.T) {
//	cli1 := newGrpcConn("localhost:9090")
//
//	serviceName, err := cli1.Register(context.Background(), NAME, ADDRESS, 10)
//	if err != nil {
//		log.Fatalln(err)
//	}
//	defer cli1.Close(context.Background(), serviceName, ADDRESS)
//
//	ticker := time.NewTicker(time.Second * 3)
//	defer ticker.Stop()
//	for {
//		select {
//		case <-ticker.C:
//			addr, err := cli1.Discover(context.Background(), NAME)
//			if err != nil {
//				log.Fatalln(err)
//			}
//			log.Println("get service", NAME, "in", addr)
//		}
//	}
//}

func TestNewRegistrarClient2(t *testing.T) {
	cli := NewRegistrarClient(WithService(NAME, ADDRESS), WithLeaseTime(5), WithRegistrarAddr([]string{"localhost:8080", "localhost:8081", "localhost:8082"}))
	defer cli.Close(context.Background())
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
}
