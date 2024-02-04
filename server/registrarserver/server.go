package registrarserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"github.com/ChenaLi0816/etcd-registrar/pubsub"
	"github.com/ChenaLi0816/etcd-registrar/utils"
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
	HEARTBEATOFFSET  = 1
	configExpireTime = 30
)

type EtcdRegistrarServer struct {
	pb.UnimplementedEtcdRegistrarServer
	*etcdModel
	LoadBalancer
	ServiceNum uint64
	locker     sync.Mutex
	msgChan    sync.Map
	lisAddr    string
	version    string
}

type newValueFunc func(oldValue string, funcArgs ...string) (string, bool)

var etcdServer *EtcdRegistrarServer = nil

var publisher = pubsub.NewPublisher()

func NewEtcdRegistrarServer(lisAddr string, etcdAddr []string, bal balancer) *EtcdRegistrarServer {
	if etcdServer == nil {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   etcdAddr,
			DialTimeout: 3 * time.Second,
			DialOptions: []grpc.DialOption{grpc.WithBlock()},
		})
		if err != nil {
			panic(err)
		}
		md := &etcdModel{Cli: cli}
		resp, _ := cli.Get(context.Background(), "config/version")
		if len(resp.Kvs) == 0 {
			panic("no version info")
		}
		etcdServer = &EtcdRegistrarServer{
			etcdModel:    md,
			LoadBalancer: NewBalancer(bal, md),
			ServiceNum:   0,
			locker:       sync.Mutex{},
			msgChan:      sync.Map{},
			lisAddr:      lisAddr,
			version:      strings.Split(string(resp.Kvs[0].Value), ":")[0],
		}
		_, err = etcdServer.putGetTxn(context.Background(), "config/registrar", configExpireTime, func(curAddr string, funcArgs ...string) (string, bool) {
			lisAddr := funcArgs[0]
			return strings.Trim(fmt.Sprintf("%s,%s", curAddr, lisAddr), ","), strings.Index(curAddr, lisAddr) == -1
		}, lisAddr)
		if err != nil {
			panic(err)
		}

		watCh := cli.Watch(context.Background(), "config", clientv3.WithPrefix())
		go etcdServer.watchConfig(watCh)
	}
	return etcdServer
}

func (s *EtcdRegistrarServer) putGetTxn(ctx context.Context, key string, expireTime int64, newValue newValueFunc, funcArgs ...string) (string, error) {
	oldValue := ""
	newV, _ := newValue(oldValue, funcArgs...)

	lres, err := s.Cli.Grant(ctx, expireTime)
	if err != nil {
		return "", err
	}
	leaseID := lres.ID

	resp, err := s.Cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), ">", 0)).
		Then(clientv3.OpGet(key)).
		Else(clientv3.OpPut(key, newV, clientv3.WithLease(leaseID))).Commit()
	if err != nil {
		return "", err
	}
	if !resp.Succeeded {
		return oldValue, nil
	}
	s.Cli.Revoke(ctx, leaseID)

	oldValue = string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
	leaseID = clientv3.LeaseID(resp.Responses[0].GetResponseRange().Kvs[0].Lease)

	for {
		newV, ok := newValue(oldValue, funcArgs...)
		if !ok {
			break
		}
		resp, err := s.Cli.Txn(ctx).
			If(clientv3.Compare(clientv3.Value(key), "=", oldValue)).
			Then(clientv3.OpPut(key, newV, clientv3.WithLease(leaseID))).
			Else(clientv3.OpGet(key)).Commit()
		if err != nil {
			return "", err
		}
		if resp.Succeeded {
			break
		}
		oldValue = string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
	}
	return oldValue, nil
}

func (s *EtcdRegistrarServer) watchConfig(ch clientv3.WatchChan) {
	for resp := range ch {
		event := resp.Events[0]
		t := strings.TrimPrefix(string(event.Kv.Key), "config/")
		if event.Type == clientv3.EventTypePut {
			v := string(event.Kv.Value)
			switch t {
			case "registrar":
				publisher.Publish("config", "registrar:"+v)
			case "version":

				ver := v
				check := false
				if i := strings.Index(v, ":"); i != -1 {
					ver = v[:i]
					opt := v[i+1:]
					if opt == "check" {
						check = true
					}
				}
				s.version = ver
				if check {
					publisher.Publish("config", "version:"+ver)
				}
			}
		} else {
			// Type Delete
			switch t {
			case "registrar":
				_, err := etcdServer.putGetTxn(context.Background(), "config/registrar", configExpireTime, func(curAddr string, funcArgs ...string) (string, bool) {
					lisAddr := funcArgs[0]
					return strings.Trim(fmt.Sprintf("%s,%s", curAddr, lisAddr), ","), strings.Index(curAddr, lisAddr) == -1
				}, s.lisAddr)
				if err != nil {
					panic(err)
				}
			case "version":

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
	name = fmt.Sprintf("%s/%s/%d_%d", req.GetVersion(), name, time.Now().Unix(), s.ServiceNum)
	s.locker.Unlock()

	err = s.newService(ctx, &serviceParams{
		name:      name,
		addr:      addr,
		leaseTime: leaseTime,
		weight:    req.GetWeight(),
		version:   req.GetVersion(),
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
		for len(ch) != 0 {
			info = utils.InfoJoin(info, <-ch)
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
				for len(ch) != 0 {
					info = utils.InfoJoin(info, <-ch)
					log.Println("fetch info:", info)
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
	_, addr, err := s.selectService(ctx, fmt.Sprintf("%s/%s", s.version, name))
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
	svcname, addr, err := s.selectService(bctx, fmt.Sprintf("%s/%s", s.version, name))
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
