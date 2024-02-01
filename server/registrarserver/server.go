package registrarserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"github.com/ChenaLi0816/etcd-registrar/pubsub"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	HEARTBEATOFFSET = 1
)

type EtcdRegistrarServer struct {
	pb.UnimplementedEtcdRegistrarServer
	*etcdModel
	LoadBalancer
	ServiceNum uint64
	locker     sync.Mutex
	msgChan    sync.Map
}

var etcdServer *EtcdRegistrarServer = nil

var publisher = pubsub.NewPublisher()

func NewEtcdRegistrarServer(lisAddr string, etcdAddr string, bal balancer) *EtcdRegistrarServer {
	if etcdServer == nil {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{etcdAddr},
			DialTimeout: 3 * time.Second,
			DialOptions: []grpc.DialOption{grpc.WithBlock()},
		})
		if err != nil {
			panic(err)
		}
		md := &etcdModel{Cli: cli}
		etcdServer = &EtcdRegistrarServer{
			etcdModel:    md,
			LoadBalancer: NewBalancer(bal, md),
			ServiceNum:   0,
			locker:       sync.Mutex{},
			msgChan:      sync.Map{},
		}
		watCh := cli.Watch(context.Background(), "config", clientv3.WithPrefix())
		go etcdServer.watchConfig(watCh)

		curAddr := ""
		get, err := cli.Get(context.Background(), "config/registrar")
		if err != nil {
			panic(err)
		}
		if len(get.Kvs) == 0 {
			cli.Put(context.Background(), "config/registrar", curAddr)
		} else {
			curAddr = string(get.Kvs[0].Value)
		}
		if strings.Index(curAddr, lisAddr) == -1 {
			for {
				resp, err := cli.Txn(context.Background()).
					If(clientv3.Compare(clientv3.Value("config/registrar"), "=", curAddr)).
					Then(clientv3.OpPut("config/registrar", strings.Trim(fmt.Sprintf("%s,%s", curAddr, lisAddr), ","))).
					Else(clientv3.OpGet("config/registrar")).Commit()
				if err != nil {
					panic(err)
				}
				if resp.Succeeded {
					break
				}
				curAddr = string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
			}
		}
	}
	return etcdServer
}

func (s *EtcdRegistrarServer) watchConfig(ch clientv3.WatchChan) {
	for resp := range ch {
		event := resp.Events[0]
		if event.Type == clientv3.EventTypePut {
			t := strings.TrimPrefix(string(event.Kv.Key), "config/")
			if t == "registrar" {
				publisher.Publish("config", "registrar:"+string(event.Kv.Value))
			}
		}
	}
}

func (s *EtcdRegistrarServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	name := req.GetName()
	addr := req.GetAddress()
	if name == "" || addr == "" {
		return nil, errors.New("string err: name or address is null")
	}
	v, leaseID, err := s.getKeyByValue(ctx, name, addr)
	if err == nil {
		if _, err := s.Cli.KeepAliveOnce(ctx, leaseID); err != nil {
			return nil, fmt.Errorf("keepalive err: %w", err)
		}
		return &pb.RegisterResponse{
			ServiceName: v,
		}, nil
	}
	leaseTime := req.GetLeaseTime()
	if leaseTime < 5 {
		return nil, errors.New("lease time cannot be lower than 5")
	}
	// TODO 锁
	s.locker.Lock()
	s.ServiceNum++
	name = fmt.Sprintf("%s/%d_%d", name, time.Now().Unix(), s.ServiceNum)
	s.locker.Unlock()

	err = s.newService(ctx, &serviceParams{
		name:      name,
		addr:      addr,
		leaseTime: leaseTime,
		weight:    req.GetWeight(),
	})
	if err != nil {
		return nil, err
	}

	log.Println("put key success", name, addr)
	publisher.AddSubscriber(newConfigSubscriber(name), "config", name)
	var reAddr []string
	get, err := s.Cli.Get(context.Background(), "config/registrar")
	if err != nil || len(get.Kvs) == 0 {
		reAddr = nil
	} else {
		reAddr = strings.Split(string(get.Kvs[0].Value), ",")
	}

	return &pb.RegisterResponse{
		ServiceName:   name,
		RegistrarAddr: reAddr,
	}, nil
}

func (s *EtcdRegistrarServer) Logout(ctx context.Context, svc *pb.Service) (*pb.Reply, error) {
	name := svc.GetName()
	addr := svc.GetAddress()
	if err := s.deleteKey(ctx, name, addr); err != nil {
		log.Println("logout err:", err)
		return nil, err
	}
	log.Println(name, addr, "logout.")
	publisher.RemoveSubscriber(name, "config")
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

	info := ""
	c, ok := s.msgChan.Load(name)
	if ok {
		ch := c.(chan string)
		if len(ch) != 0 {
			info = <-ch
			log.Println("fetch info:", info)
		}
	}
	return &pb.Reply{Info: info}, nil
}

func (s *EtcdRegistrarServer) HeartbeatPassive(stream pb.EtcdRegistrar_HeartbeatPassiveServer) error {
	rec, err := stream.Recv()
	if err != nil {
		log.Println("heartbeat passive recv err:", err)
		return err
	}
	name := rec.GetName()
	_, leaseID, err := s.getValueByKey(context.Background(), name)
	if err != nil {
		log.Println("get value err:", err)
		return err
	}
	resp, err := s.Cli.KeepAliveOnce(context.Background(), leaseID)
	if err != nil {
		log.Println("keep alive err:", err)
		return err
	}
	ticker := time.NewTicker(time.Duration(resp.TTL-HEARTBEATOFFSET) * time.Second)
	defer func() {
		ticker.Stop()
		publisher.RemoveSubscriber(name, "config")
	}()

	for {
		select {
		case <-ticker.C:
			info := ""
			c, ok := s.msgChan.Load(name)
			if ok {
				ch := c.(chan string)
				if len(ch) != 0 {
					info = <-ch
				}
			}
			err = stream.Send(&pb.Reply{Info: info})
			if err != nil {
				log.Println("heartbeat passive send err:", err)
				return err
			}
			_, err = stream.Recv()
			if err != nil {
				if err == io.EOF {
					log.Println("stream read EOF")
					return nil
				}
				log.Println("heartbeat passive recv err:", err)
				return err
			}
			_, err = s.Cli.KeepAliveOnce(context.Background(), leaseID)
			if err != nil {
				log.Println("keep alive err:", err)
				return err
			}
		}
		log.Println("heatbeat passive success", name)
	}
}

func (s *EtcdRegistrarServer) Discover(ctx context.Context, req *pb.DiscoverRequest) (*pb.DiscoverResponse, error) {
	name := req.GetName()
	_, addr, err := s.selectService(ctx, name)
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
	ip, _ := net.ResolveTCPAddr("tcp", req.GetLocalAddr())
	p := &peer.Peer{
		Addr:      ip,
		LocalAddr: nil,
		AuthInfo:  nil,
	}
	bctx := peer.NewContext(context.Background(), p)
choose:
	svcname, addr, err := s.selectService(bctx, name)
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
