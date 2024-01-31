package registrarclient

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

const (
	NAME     = "test"
	ADDRESS1 = "127.0.0.1:2221"
	ADDRESS2 = "127.0.0.1:2222"
	ADDRESS3 = "127.0.0.1:2223"
)

func TestNewRegistrarClient(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080", "localhost:8081", "localhost:8082"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
	time.Sleep(time.Second * 30)
}

func TestNewRegistrarClient2(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
	time.Sleep(time.Second * 30)
}

func TestRegistrarClient_Discover(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
}

func TestNilChannel(t *testing.T) {
	select {
	case <-context.Background().Done():
		log.Println("backendground done")
	default:
		log.Println("nothing happen")
	}
}

func closeAfter(cli RegistrarClient, t time.Duration) {
	time.Sleep(t)
	cli.Close()
}

func TestSubscribe(t *testing.T) {
	cli1 := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1).WithRegistrarAddress([]string{"localhost:8080"}))
	go closeAfter(cli1, time.Second*5)
	err := cli1.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	cli2 := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS2).WithRegistrarAddress([]string{"localhost:8080"}))
	go closeAfter(cli2, time.Second*10)
	err = cli2.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	cli3 := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS3).WithRegistrarAddress([]string{"localhost:8080"}))
	go closeAfter(cli3, time.Second*15)
	err = cli3.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	cli := NewRegistrarClient(NewDefaultOptions().WithRegistrarAddress([]string{"localhost:8080"}))
	defer cli.Close()
	ch, err := cli.Subscribe(context.Background(), NAME)
	if err != nil {
		log.Fatalln(err)
	}
	for resp := range ch {
		fmt.Printf("available %v addr %v\n", resp.Available, resp.Addr)
	}
}

func TestGrpcConn(t *testing.T) {
	c := &basicClient{}
	err := c.newGrpcConn("127.0.0.1:8080")
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println("success")
	}
}

func TestClose(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	//time.Sleep(time.Second)
}

func TestPassiveClient(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080", "localhost:8081", "localhost:8082"}).WithPassive(true))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
}
