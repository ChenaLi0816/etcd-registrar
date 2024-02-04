package registrarclient

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	NAME     = "test"
	ADDRESS1 = "127.0.0.1:2221"
	VERSION1 = "1"
	VERSION2 = "2"
	VERSION3 = "3"
	ADDRESS2 = "127.0.0.1:2222"
	ADDRESS3 = "127.0.0.1:2223"
)

func TestNewRegistrarClient(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080", "localhost:8081", "localhost:8082"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	//go func() {
	//	t := time.NewTicker(time.Second * 2)
	//	defer t.Stop()
	//	for i := 0; i < 100; i++ {
	//		select {
	//		case <-t.C:
	//			resp, err := cli.Discover(context.Background(), NAME)
	//			if err != nil {
	//				log.Println("discover err:", err)
	//				continue
	//			}
	//			fmt.Println("discover in", resp)
	//		}
	//	}
	//}()
	select {}
	time.Sleep(time.Second * 30)
}

func TestNewRegistrarClient2(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS2, VERSION1).WithWeight(3).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080", "localhost:8081", "localhost:8082"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	//go func() {
	//	t := time.NewTicker(time.Second * 2)
	//	defer t.Stop()
	//	for i := 0; i < 100; i++ {
	//		select {
	//		case <-t.C:
	//			resp, err := cli.Discover(context.Background(), NAME)
	//			if err != nil {
	//				log.Println("discover err:", err)
	//				continue
	//			}
	//			fmt.Println("discover in", resp)
	//		}
	//	}
	//}()
	select {}
	time.Sleep(time.Second * 30)
}

func TestNewRegistrarClient3(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS3, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	go func() {
		t := time.NewTicker(time.Second * 2)
		defer t.Stop()
		for i := 0; i < 100; i++ {
			select {
			case <-t.C:
				resp, err := cli.Discover(context.Background(), NAME)
				if err != nil {
					log.Println("discover err:", err)
					continue
				}
				fmt.Println("discover in", resp)
			}
		}
	}()
	select {}
	//time.Sleep(time.Second * 30)
}

func TestRegistrarClient_Discover(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"192.168.1.7:8080"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
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
	cli1 := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithRegistrarAddress([]string{"localhost:8080"}))
	go closeAfter(cli1, time.Second*5)
	err := cli1.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	cli2 := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS2, VERSION1).WithRegistrarAddress([]string{"localhost:8080"}))
	go closeAfter(cli2, time.Second*10)
	err = cli2.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	cli3 := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS3, VERSION1).WithRegistrarAddress([]string{"localhost:8080"}))
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

func TestActiveClientClose(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	//select {}
	time.Sleep(time.Second * 20)
}

func TestPassiveClient(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080", "localhost:8081"}).WithPassive(true))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
}

func TestPassiveClientClose(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}).WithPassive(true))
	defer cli.Close()
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	time.Sleep(time.Second * 20)
}

func TestCloseChan(t *testing.T) {
	ch := make(chan struct{})
	close(ch)
	fmt.Println(ch == closeChan)
}

func TestConfigNotice(t *testing.T) {
	cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"127.0.0.1:8080"}))
	defer cli.Close()
	time.Sleep(time.Second * 2)
	err := cli.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
	time.Sleep(time.Second * 15)
}

var wrong = atomic.Int32{}

func TestMaxEtcdConnect(t *testing.T) {
	const MaxConnect = 1000
	const etcdAddr = "127.0.0.1:2379"
	wrong.Store(0)
	wg := sync.WaitGroup{}
	wg.Add(MaxConnect)
	for i := 0; i < MaxConnect; i++ {
		i := i
		go func() {
			defer wg.Done()
			cli, err := clientv3.New(clientv3.Config{
				Endpoints:   []string{etcdAddr},
				DialTimeout: 3 * time.Second,
				DialOptions: []grpc.DialOption{grpc.WithBlock()},
			})
			if err != nil {
				fmt.Println(i, "err:", err)
				wrong.Add(1)
				return
			}
			defer cli.Close()
			resp, _ := cli.Grant(context.Background(), 5)
			cli.Put(context.Background(), fmt.Sprintf("k%d", i), "ok", clientv3.WithLease(resp.ID))
			fmt.Println(i, "put ok")

			time.Sleep(time.Second * 10)
		}()
	}
	wg.Wait()
	fmt.Println("err cli:", wrong.Load())
	time.Sleep(time.Second * 3)
}

func TestMaxRegistrarConnect(t *testing.T) {
	const MaxConnect = 10
	wrong.Store(0)
	wg := sync.WaitGroup{}
	wg.Add(MaxConnect)
	for i := 0; i < MaxConnect; i++ {
		i := i
		go func() {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					fmt.Println(i, "err:", err)
					wrong.Add(1)
				}
			}()
			cli := NewRegistrarClient(NewDefaultOptions().WithService(NAME, ADDRESS1, VERSION1).WithLeaseTime(5).WithRegistrarAddress([]string{"localhost:8080"}).WithPassive(true))
			defer cli.Close()
			err := cli.Register(context.Background())
			if err != nil {
				log.Println(i, "resg err:", err)
				return
			}
			log.Println(i, "resg suc")
			time.Sleep(time.Second * 10)
			//
		}()
	}
	wg.Wait()
	fmt.Println("err:", wrong.Load())
	time.Sleep(time.Second * 3)
}

func TestDefer(t *testing.T) {

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("err:", err)
		}
	}()
	panic("err")
	defer fmt.Println(1)
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context done:", ctx.Err())
				return
			}
		}
	}(ctx)
	time.Sleep(time.Second * 3)
	cancel()
	time.Sleep(time.Second)
}

func TestEtcdConn(t *testing.T) {
	const etcdAddr = "127.0.0.1:4379"
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 3 * time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer cli.Close()
	//_, err = cli.Put(context.Background(), "config/version", "2")
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	get, err := cli.Get(context.Background(), "key")
	if len(get.Kvs) == 0 {
		log.Println("len 0")
		return
	}
	fmt.Println("key:", string(get.Kvs[0].Key), "va:", string(get.Kvs[0].Value))
	fmt.Println("ok")
}
