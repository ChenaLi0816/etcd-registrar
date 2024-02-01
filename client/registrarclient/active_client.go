package registrarclient

import (
	"context"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"log"
	"os"
	"strings"
	"time"
)

type activeClient struct {
	basicClient

	ticker    *time.Ticker
	closeChan chan struct{}
}

func (c *activeClient) Register(ctx context.Context) error {
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

	if resp.RegistrarAddr != nil {
		log.Println("get addr:", resp.RegistrarAddr)
		c.options.address = resp.RegistrarAddr
	}

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
			return
		case <-c.ticker.C:
			resp, err := c.cli.HeartbeatActive(ctx, &pb.Service{
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
				err = c.Register(ctx)
				if err != nil {
					log.Println("retry register err:", err)
				}
				return
			}
			info := resp.GetInfo()
			if info != "" {
				log.Println("get info:", info)
				i := strings.Index(info, ":")
				t, msg := info[:i], info[i+1:]
				switch t {
				case "registrar":
					c.options.address = strings.Split(msg, ",")
				}
			}
			log.Println(c.uniqueID, c.options.localAddr, "heartbeat success")

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
