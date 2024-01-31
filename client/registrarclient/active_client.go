package registrarclient

import (
	"context"
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"log"
	"os"
	"time"
)

type activeClient struct {
	basicClient

	ticker    *time.Ticker
	closeChan chan struct{}
}

func (c *activeClient) Register(ctx context.Context) error {
	if c.ticker != nil {
		return fmt.Errorf("service time ticker is not nil")
	}
	if c.options.name == "" || c.options.localAddr == "" {
		panic("service name or service address is null")
	}
	req := &pb.RegisterRequest{
		Name:      c.options.name,
		Address:   c.options.localAddr,
		LeaseTime: c.options.leaseTime,
	}
	resp, err := c.cli.Register(ctx, req)
	if err != nil {
		log.Println("register err:", err, "now retry")
		err = c.switchConnection(ctx)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
	c.uniqueID = resp.GetServiceName()
	c.ticker = time.NewTicker(time.Second * time.Duration(c.options.leaseTime-HEARTBEATOFFSET))
	c.closeChan = make(chan struct{}, 1)
	log.Println("register success")
	go c.timeTick(ctx)
	return nil
}

func (c *activeClient) switchConnection(ctx context.Context) error {
	c.close(true)
	return c.basicClient.switchConnection(ctx)
}

func (c *activeClient) timeTick(ctx context.Context) {
	for {
		select {
		case <-c.closeChan:
			log.Println(c.uniqueID, "closeChan time tick")
			c.ticker = nil
			return
		case <-c.ticker.C:
			_, err := c.cli.HeartbeatActive(ctx, &pb.Service{
				Name:    c.uniqueID,
				Address: c.options.localAddr,
			})
			if err != nil {
				log.Println("heartbeat err:", err, "switch connection")
				err = c.switchConnection(ctx)
				if err != nil {
					log.Println(err)
					os.Exit(1)
				}
				c.ticker = nil
				err = c.Register(ctx)
				if err != nil {
					log.Println("retry register err:", err)
				}
				return
			} else {
				log.Println(c.uniqueID, c.options.localAddr, "heartbeat success")
			}
		}
	}
}

func (c *activeClient) logout(ctx context.Context) error {
	_, err := c.cli.Logout(ctx, &pb.Service{
		Name:    c.uniqueID,
		Address: c.options.localAddr,
	})
	return err
}

// TODO new了但没注册
func (c *activeClient) close(reason bool) {
	if c.ticker != nil {
		close(c.closeChan)
		c.ticker.Stop()
	}
	c.basicClient.close(reason)
}

func (c *activeClient) Close() {
	c.close(false)
}
