package utils

import (
	"context"
	"errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	ErrLockOccupied = errors.New("lock is occupied")
)

func AcquireLock(cli *clientv3.Client, lockKey string, ttl int64) (clientv3.LeaseID, error) {
	// 创建租约
	leaseResp, err := cli.Grant(context.TODO(), ttl)
	if err != nil {
		return 0, err
	}

	// 尝试创建锁
	txnResp, err := cli.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseResp.ID))).
		Commit()
	if err != nil {
		return 0, err
	}

	// 判断锁是否被成功创建
	if !txnResp.Succeeded {
		return 0, ErrLockOccupied
	}

	return leaseResp.ID, nil
}

func ReleaseLock(cli *clientv3.Client, leaseID clientv3.LeaseID) {
	cli.Revoke(context.TODO(), leaseID)
}
