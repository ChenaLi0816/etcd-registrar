package registrarserver

import (
	"context"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	ErrKeyNoExist   = errors.New("key don't exist")
	ErrValueNoExist = errors.New("value don't exist")
)

type etcdModel struct {
	Cli *clientv3.Client
}

func (s *etcdModel) putKeyWithTime(ctx context.Context, key, value string, leaseTime int64) (clientv3.LeaseID, error) {
	res, err := s.Cli.Grant(ctx, leaseTime)
	if err != nil {
		return 0, fmt.Errorf("grant err: %w", err)
	}
	leaseID := res.ID
	_, err = s.Cli.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		return 0, fmt.Errorf("put err: %w", err)
	}
	return leaseID, nil
}

func (s *etcdModel) putKeyWithLeaseID(ctx context.Context, key, value string, leaseID clientv3.LeaseID) error {
	_, err := s.Cli.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		return fmt.Errorf("put err: %w", err)
	}
	return nil
}

func (s *etcdModel) getValueByKey(ctx context.Context, key string) (string, clientv3.LeaseID, error) {
	resp, err := s.Cli.Get(ctx, key)
	if err != nil {
		return "", 0, err
	}
	if len(resp.Kvs) == 0 {
		return "", 0, ErrKeyNoExist
	}
	return string(resp.Kvs[0].Value), clientv3.LeaseID(resp.Kvs[0].Lease), nil
}

func (s *etcdModel) getKeyByValue(ctx context.Context, prefix, value string) (string, clientv3.LeaseID, error) {
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

func (s *etcdModel) deleteKey(ctx context.Context, key, value string) error {
	getv, leaseID, err := s.getValueByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("get key %s err:%w", key, err)
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
