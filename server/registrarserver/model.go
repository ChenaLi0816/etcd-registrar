package registrarserver

import (
	"context"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"math/rand"
)

var (
	ErrKeyNoExist   = errors.New("key don't exist")
	ErrValueNoExist = errors.New("value don't exist")
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
	resp, err := s.Cli.Get(ctx, key)
	if err != nil {
		return "", 0, err
	}
	if len(resp.Kvs) == 0 {
		return "", 0, ErrKeyNoExist
	}
	return string(resp.Kvs[0].Value), clientv3.LeaseID(resp.Kvs[0].Lease), nil
}

func (s *EtcdRegistrarServer) getKeyByValue(ctx context.Context, prefix, value string) (string, clientv3.LeaseID, error) {
	resp, err := s.Cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return "", 0, err
	}
	if len(resp.Kvs) == 0 {
		return "", 0, ErrKeyNoExist
	}
	for _, v := range resp.Kvs {
		if string(v.Value) == value {
			return string(v.Key), clientv3.LeaseID(v.Lease), nil
		}
	}
	return "", 0, ErrValueNoExist
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

//func (s *EtcdRegistrarServer) getValuesByKey(ctx context.Context, prefix string) (string, error) {
//	resp, err := s.Cli.Get(ctx, prefix, clientv3.WithPrefix())
//	if err != nil {
//		return "", fmt.Errorf("get err:%w", err)
//	}
//	if len(resp.Kvs) == 0 {
//		return "", fmt.Errorf("key %s don't exist", prefix)
//	}
//
//}

func (s *EtcdRegistrarServer) chooseService(ctx context.Context, prefix string) (string, string, error) {
	resp, err := s.Cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return "", "", fmt.Errorf("get err:%w", err)
	}
	if len(resp.Kvs) == 0 {
		return "", "", ErrKeyNoExist
	}
	idx := s.average(len(resp.Kvs))
	return string(resp.Kvs[idx].Key), string(resp.Kvs[idx].Value), nil
}

// TODO 负载均衡算法
func (s *EtcdRegistrarServer) average(num int) int {
	return rand.Int() % num
}
