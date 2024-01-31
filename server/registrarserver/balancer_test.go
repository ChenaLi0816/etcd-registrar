package registrarserver

import (
	"context"
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/peer"
	"log"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

func runBalancer(cli *clientv3.Client, tp balancer, serviceNum int, selTime int, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	blc := NewBalancer(tp, &etcdModel{Cli: cli})
	for i := 0; i < serviceNum; i++ {
		err := blc.newService(context.Background(), &serviceParams{
			name:      fmt.Sprintf("test/%d", i),
			addr:      fmt.Sprintf("localhost:909%d", i),
			leaseTime: int64((i + 1) * 50),
			weight:    int32(i + 1),
		})
		if err != nil {
			log.Println(err)
		}
	}
	ip, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:"+strconv.Itoa(int(utils.RandInt(8000, false))))
	p := &peer.Peer{
		Addr:      ip,
		LocalAddr: nil,
		AuthInfo:  nil,
	}
	fmt.Println("local addr:", p.Addr.String())
	ctx := peer.NewContext(context.Background(), p)
	ticker := time.NewTicker(time.Second)
	for i := 0; i < selTime; i++ {
		select {
		case <-ticker.C:
			name, addr, err := blc.selectService(ctx, "test")
			if err != nil {
				log.Println(err)
				continue
			}
			fmt.Printf("select service %s in %s\n", name, addr)
		}
	}

}

func TestRandomBalancer(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer cli.Close()
	runBalancer(cli, RandomBalancer, 5, 10, nil)
}

func TestRoundRobin(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer cli.Close()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go runBalancer(cli, RoundRobin, 5, 20, &wg)
	go runBalancer(cli, RoundRobin, 5, 20, &wg)
	wg.Wait()
}

func TestWeightRobin(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer cli.Close()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go runBalancer(cli, WeightRoundRobin, 5, 15, &wg)
	go runBalancer(cli, WeightRoundRobin, 5, 15, &wg)
	wg.Wait()
}

func TestIpHash(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer cli.Close()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go runBalancer(cli, IpHashBalancer, 5, 15, &wg)
	go runBalancer(cli, IpHashBalancer, 5, 15, &wg)
	wg.Wait()
}
