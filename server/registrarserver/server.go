package registrarserver

import (
	"context"
	"errors"
	"etcd-registrar/proto/pb"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"sync"
	"time"
)

type EtcdRegistrarServer struct {
	pb.UnimplementedEtcdRegistrarServer
	Cli        *clientv3.Client
	ServiceNum uint64
	locker     sync.Mutex
}

var etcdServer *EtcdRegistrarServer = nil

func NewEtcdRegistrarServer(addr string) *EtcdRegistrarServer {
	if etcdServer == nil {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{addr},
			DialTimeout: 3 * time.Second,
		})
		if err != nil {
			log.Fatalln(err)
		}
		etcdServer = &EtcdRegistrarServer{
			Cli:        cli,
			ServiceNum: 0,
			locker:     sync.Mutex{},
		}
	}
	return etcdServer
}

func (s *EtcdRegistrarServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	name := req.GetName()
	addr := req.GetAddress()
	if name == "" || addr == "" {
		return nil, errors.New("string err: name or address is null")
	}
	v, leaseID, err := s.getKeyByValue(ctx, name, addr)
	// TODO 分布式锁，有可能两边读的时候都没有，写的时候都写了
	if err == nil {
		if _, err := s.Cli.KeepAliveOnce(ctx, leaseID); err != nil {
			return nil, fmt.Errorf("keepalive err: %w", err)
		}
		return &pb.RegisterResponse{
			ServiceName: v,
		}, nil
	}
	// TODO 锁
	s.locker.Lock()
	s.ServiceNum++
	name = fmt.Sprintf("%s/%d_%d", name, time.Now().Unix(), s.ServiceNum)
	s.locker.Unlock()
	if err := s.putKey(ctx, name, addr, req.GetLeaseTime()); err != nil {
		return nil, err
	}
	log.Println("put key success", name, addr)

	return &pb.RegisterResponse{
		ServiceName: name,
	}, nil
}

func (s *EtcdRegistrarServer) Logout(ctx context.Context, svc *pb.Service) (*pb.Reply, error) {
	name := svc.GetName()
	addr := svc.GetAddress()
	if err := s.deleteKey(ctx, name, addr); err != nil {
		return nil, err
	}
	log.Println(name, addr, "logout.")
	return &pb.Reply{}, nil
}

func (s *EtcdRegistrarServer) HeartbeatActive(ctx context.Context, svc *pb.Service) (*pb.Reply, error) {
	name := svc.GetName()
	addr := svc.GetAddress()
	getv, leaseID, err := s.getValueByKey(ctx, name)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if addr != getv {
		log.Println("match err: address", addr, "don't match", getv)
		return nil, errors.New("match err: address don't match")
	}
	if _, err := s.Cli.KeepAliveOnce(ctx, leaseID); err != nil {
		return nil, fmt.Errorf("keepalive err: %w", err)
	}
	log.Println(name, "in", addr, "keepalive success")
	return &pb.Reply{}, nil
}

func (s *EtcdRegistrarServer) Discover(ctx context.Context, req *pb.DiscoverRequest) (*pb.DiscoverResponse, error) {
	name := req.GetName()
	_, addr, err := s.chooseService(ctx, name)
	if err != nil {
		log.Println("discover err:", err)
		return nil, err
	}
	log.Println("service", name, "discover in", addr)
	return &pb.DiscoverResponse{
		Address: addr,
	}, nil
}

func (s *EtcdRegistrarServer) Subscribe(req *pb.SubscribeRequest, stream pb.EtcdRegistrar_SubscribeServer) error {
	name := req.GetName()
choose:
	svcname, addr, err := s.chooseService(context.Background(), name)
	if err != nil {
		log.Println("subscribe err:", err)
		err = stream.Send(&pb.SubscribeResponse{
			Available: false,
			Addr:      "",
		})
		if err != nil {
			log.Println("send err:", err)
		}
		return err
	}
	err = stream.Send(&pb.SubscribeResponse{
		Available: true,
		Addr:      addr,
	})
	if err != nil {
		log.Println("send err:", err)
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	watchChan := s.Cli.Watch(ctx, svcname)

	for resp := range watchChan {
		event := resp.Events[0]
		if event.Type == clientv3.EventTypeDelete {
			cancel()
			goto choose
		}
		addr = string(event.Kv.Value)
		err = stream.Send(&pb.SubscribeResponse{
			Available: true,
			Addr:      addr,
		})
		if err != nil {
			log.Println("send err:", err)
			cancel()
			return err
		}
	}
	cancel()
	return ErrKeyNoExist
}
