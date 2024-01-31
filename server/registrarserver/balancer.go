package registrarserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/peer"
	"strconv"
	"strings"
	"time"
)

type balancer int

const (
	RoundRobin balancer = iota
	WeightRoundRobin
	RandomBalancer
	IpHashBalancer
)

var (
	ErrContextPeerNoExist = errors.New("peer info isn't exist in context")
)

type serviceParams struct {
	name      string
	addr      string
	leaseTime int64
	weight    int32
}

type LoadBalancer interface {
	selectService(ctx context.Context, name string) (string, string, error)
	newService(ctx context.Context, p *serviceParams) error
}

func NewBalancer(tp balancer, md *etcdModel) LoadBalancer {
	switch tp {
	case RoundRobin:
		return &roundRobin{md}
	case WeightRoundRobin:
		return &weightRoundRobin{md}
	case RandomBalancer:
		return &randomBalancer{md}
	case IpHashBalancer:
		return &ipHashBalancer{md}
	default:
		return &roundRobin{md}
	}
}

type roundRobin struct {
	*etcdModel
}

func (r *roundRobin) selectService(ctx context.Context, name string) (string, string, error) {
	get, err := r.Cli.Get(ctx, name, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return "", "", err
	}
	maxIndex := len(get.Kvs)
	if maxIndex == 0 {
		return "", "", ErrKeyNoExist
	}
	curIndex, _, err := r.getValueByKey(ctx, "curIndex")
	if err != nil {
		if err != ErrKeyNoExist {
			return "", "", err
		}
		curIndex = "0"
		_, err = r.Cli.Put(ctx, "curIndex", curIndex)
		if err != nil {
			return "", "", err
		}
	}
	for {
		txnResponse, err := r.Cli.Txn(ctx).
			If(clientv3.Compare(clientv3.Value("curIndex"), "=", curIndex)).
			Then(clientv3.OpPut("curIndex", utils.StringModAdd(curIndex, 1, maxIndex))).
			Else(clientv3.OpGet("curIndex")).Commit()
		if err != nil {
			return "", "", err
		}
		if txnResponse.Succeeded {
			break
		}
		curIndex = string(txnResponse.Responses[0].GetResponseRange().Kvs[0].Value)
	}

	index, _ := strconv.Atoi(curIndex)
	index %= maxIndex
	return string(get.Kvs[index].Key), string(get.Kvs[index].Value), nil
}

func (r *roundRobin) newService(ctx context.Context, p *serviceParams) error {
	_, err := r.putKeyWithTime(ctx, p.name, p.addr, p.leaseTime)
	return err
}

type weightRoundRobin struct {
	*etcdModel
}

type weightServer struct {
	weight    int
	curWeight int
	name      string
	leaseID   clientv3.LeaseID
}

func parseWeight(resp *clientv3.GetResponse) []*weightServer {
	s := make([]*weightServer, 0, len(resp.Kvs))
	for _, it := range resp.Kvs {
		name := strings.TrimPrefix(string(it.Key), "weight/")
		w := strings.Split(string(it.Value), ":")
		weight, _ := strconv.Atoi(w[0])
		curWeight, _ := strconv.Atoi(w[1])
		s = append(s, &weightServer{
			weight:    weight,
			curWeight: curWeight,
			name:      name,
			leaseID:   clientv3.LeaseID(it.Lease),
		})

	}

	return s
}

func (r *weightRoundRobin) selectService(ctx context.Context, name string) (string, string, error) {
	var lock clientv3.LeaseID
	var err error
	for {
		lock, err = utils.AcquireLock(r.Cli, "weightLock/"+name, 1)
		if err == nil {
			break
		}
		if err != utils.ErrLockOccupied {
			return "", "", err
		}
		time.Sleep(time.Millisecond * 10)
	}
	defer utils.ReleaseLock(r.Cli, lock)

	get, err := r.Cli.Get(ctx, "weight/"+name, clientv3.WithPrefix())
	if len(get.Kvs) == 0 {
		return "", "", ErrKeyNoExist
	}
	svs := parseWeight(get)
	total := 0
	best := svs[0]

	for _, s := range svs {
		s.curWeight += s.weight
		total += s.weight
		if best.curWeight < s.curWeight {
			best = s
		}
	}
	best.curWeight -= total
	for _, s := range svs {
		r.putKeyWithLeaseID(ctx, "weight/"+s.name, fmt.Sprintf("%d:%d", s.weight, s.curWeight), s.leaseID)
	}

	get, err = r.Cli.Get(ctx, best.name)
	if err != nil {
		return "", "", err
	}
	// TODO 选择的瞬间刚好过期
	return string(get.Kvs[0].Key), string(get.Kvs[0].Value), nil
}

func (r *weightRoundRobin) newService(ctx context.Context, p *serviceParams) error {
	leaseID, err := r.putKeyWithTime(ctx, p.name, p.addr, p.leaseTime)
	if err != nil {
		return err
	}
	if p.weight == 0 {
		p.weight = 1
	}
	err = r.putKeyWithLeaseID(ctx, "weight/"+p.name, strconv.Itoa(int(p.weight))+":0", leaseID)
	if err != nil {
		return err
	}
	return nil
}

type randomBalancer struct {
	*etcdModel
}

func (r *randomBalancer) selectService(ctx context.Context, name string) (string, string, error) {
	get, err := r.Cli.Get(ctx, name, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return "", "", fmt.Errorf("get err: %w", err)
	}
	maxIndex := len(get.Kvs)
	if maxIndex == 0 {
		return "", "", ErrKeyNoExist
	}
	index := utils.RandInt(int64(maxIndex), false)
	return string(get.Kvs[index].Key), string(get.Kvs[index].Value), nil
}

func (r *randomBalancer) newService(ctx context.Context, p *serviceParams) error {
	_, err := r.putKeyWithTime(ctx, p.name, p.addr, p.leaseTime)
	return err
}

type ipHashBalancer struct {
	*etcdModel
}

func (r *ipHashBalancer) selectService(ctx context.Context, name string) (string, string, error) {
	get, err := r.Cli.Get(ctx, name, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return "", "", err
	}
	maxIndex := uint32(len(get.Kvs))
	if maxIndex == 0 {
		return "", "", ErrKeyNoExist
	}
	remote, ok := peer.FromContext(ctx)
	if !ok {
		return "", "", ErrContextPeerNoExist
	}
	sourceIP := remote.Addr.String()
	index := utils.Hash(sourceIP) % maxIndex
	return string(get.Kvs[index].Key), string(get.Kvs[index].Value), nil
}

func (r *ipHashBalancer) newService(ctx context.Context, p *serviceParams) error {
	_, err := r.putKeyWithTime(ctx, p.name, p.addr, p.leaseTime)
	return err
}
