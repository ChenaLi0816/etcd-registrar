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
	v, exist := s.getKeyByValue(ctx, name, addr)
	// TODO 分布式锁，有可能两边读的时候都没有，写的时候都写了
	if exist {
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

func (s *EtcdRegistrarServer) putKey(ctx context.Context, key, value string, leaseTime int64) error {
	res, err := s.Cli.Grant(ctx, leaseTime)
	if err != nil {
		return fmt.Errorf("grant err: %w", err)
	}
	leaseID := res.ID
	_, err = s.Cli.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		return fmt.Errorf("put err: %w", err)
	}
	return nil
}

func (s *EtcdRegistrarServer) getValueByKey(ctx context.Context, key string) (string, clientv3.LeaseID, bool) {
	resp, _ := s.Cli.Get(ctx, key)
	if len(resp.Kvs) == 0 {
		return "", 0, false
	}
	return string(resp.Kvs[0].Value), clientv3.LeaseID(resp.Kvs[0].Lease), true
}

func (s *EtcdRegistrarServer) getKeyByValue(ctx context.Context, prefix, value string) (string, bool) {
	resp, _ := s.Cli.Get(ctx, prefix, clientv3.WithPrefix())
	if len(resp.Kvs) == 0 {
		return "", false
	}
	for _, v := range resp.Kvs {
		if string(v.Value) == value {
			return string(v.Key), true
		}
	}
	return "", false
}

func (s *EtcdRegistrarServer) deleteKey(ctx context.Context, key, value string) error {
	getv, leaseID, exist := s.getValueByKey(ctx, key)
	if !exist {
		return errors.New("get err: key don't exist")
	}
	if value != getv {
		return errors.New("match err: address don't match")
	}
	// 撤销租约
	if leaseID != 0 {
		_, err := s.Cli.Revoke(ctx, leaseID)
		if err != nil {
			return fmt.Errorf("revoke lease err: %w", err)
		}
	}
	// TODO 删除过程中有别的服务调用
	_, err := s.Cli.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("delete err: %w", err)
	}
	return nil
}

func (s *EtcdRegistrarServer) HeartbeatActive(ctx context.Context, svc *pb.Service) (*pb.Reply, error) {
	name := svc.GetName()
	addr := svc.GetAddress()
	getv, leaseID, exist := s.getValueByKey(ctx, name)
	if !exist {
		log.Println("get err: key", name, "don't exist")
		return nil, errors.New("get err: key don't exist")
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
