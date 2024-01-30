package utils

import (
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"sync"
	"testing"
	"time"
)

func lockGo(wg *sync.WaitGroup, lockKey string, cli *clientv3.Client, goName string) {
	defer wg.Done()
	var lock clientv3.LeaseID
	var err error
	times := 0
	for {
		lock, err = AcquireLock(cli, lockKey, 5)
		if err == nil {
			fmt.Printf("go %s get lock success\n", goName)
			break
		}
		if err != ErrLockOccupied {
			fmt.Println("err:", err)
			return
		}
		times++
		fmt.Printf("go %s spin %d\n", goName, times)
		time.Sleep(time.Millisecond * 10)
	}
	time.Sleep(time.Second)
	ReleaseLock(cli, lock)
}

func TestLock(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalln(err)
	}
	lockKey := "lock"
	wg := sync.WaitGroup{}
	wg.Add(3)
	go lockGo(&wg, lockKey, cli, "1")
	go lockGo(&wg, lockKey, cli, "2")
	go lockGo(&wg, lockKey, cli, "3")
	wg.Wait()
}
