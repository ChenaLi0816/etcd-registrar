package registrarserver

import (
	"context"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"math/rand"
)

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

func (s *EtcdRegistrarServer) getValueByKey(ctx context.Context, key string) (string, clientv3.LeaseID, error) {
	resp, _ := s.Cli.Get(ctx, key)
	if len(resp.Kvs) == 0 {
		return "", 0, fmt.Errorf("key %s don't exist", key)
	}
	return string(resp.Kvs[0].Value), clientv3.LeaseID(resp.Kvs[0].Lease), nil
}

func (s *EtcdRegistrarServer) getKeyByValue(ctx context.Context, prefix, value string) (string, clientv3.LeaseID, error) {
	resp, _ := s.Cli.Get(ctx, prefix, clientv3.WithPrefix())
	if len(resp.Kvs) == 0 {
		return "", 0, fmt.Errorf("key %s don't exist", prefix)
	}
	for _, v := range resp.Kvs {
		if string(v.Value) == value {
			return string(v.Key), clientv3.LeaseID(v.Lease), nil
		}
	}
	return "", 0, fmt.Errorf("value %s don't exist", value)
}

func (s *EtcdRegistrarServer) deleteKey(ctx context.Context, key, value string) error {
	getv, leaseID, err := s.getValueByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("get err:%w", err)
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
	_, err = s.Cli.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("delete err: %w", err)
	}
	return nil
}

func (s *EtcdRegistrarServer) chooseService(ctx context.Context, prefix string) (string, error) {
	resp, err := s.Cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return "", fmt.Errorf("get err:%w", err)
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("key %s don't exist", prefix)
	}
	num := len(resp.Kvs)
	return string(resp.Kvs[s.average(num)].Value), nil
}

// TODO 负载均衡算法
func (s *EtcdRegistrarServer) average(num int) int {
	return rand.Int() % num
}
