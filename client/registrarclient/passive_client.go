package registrarclient

import (
	"context"
	"errors"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"log"
	"os"
)

type passiveClient struct {
	basicClient

	closeChan chan struct{}
}

var closeChan chan struct{} = make(chan struct{}, 1)

func init() {
	close(closeChan)
}

func (c *passiveClient) Register(ctx context.Context) error {
	if c.options.name == "" || c.options.localAddr == "" {
		return errors.New("service name or service address is null")
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
	log.Println("register success")
	c.closeChan = make(chan struct{}, 1)
	go c.heartbeat(ctx)
	return nil
}

func (c *passiveClient) heartbeat(ctx context.Context) {
	stream, err := c.cli.HeartbeatPassive(context.Background())
	if err != nil {
		log.Println("heartbeat err:", err)
		return
	}
	defer func() {
		_ = stream.CloseSend()
		if c.closeChan != closeChan {
			log.Println("switch connection")
			err = c.switchConnection(ctx)
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
			err = c.Register(ctx)
			if err != nil {
				log.Println("register err:", err)
			}
		}
	}()

	if err = stream.Send(&pb.CheckHealth{Name: c.uniqueID}); err != nil {
		log.Println("stream send err:", err)
		return
	}
	for {
		select {
		case <-c.closeChan:
			return
		default:
			_, err = stream.Recv()
			if err != nil {
				log.Println("stream recv err:", err)
				return
			}
			if err = stream.Send(&pb.CheckHealth{}); err != nil {
				log.Println("stream send err:", err)
				return
			}
			log.Println("heartbeat passive success", c.uniqueID)
		}

	}
}

func (c *passiveClient) switchConnection(ctx context.Context) error {
	c.close(true)
	return c.basicClient.switchConnection(ctx)
}

func (c *passiveClient) close(reason bool) {
	if c.closeChan != nil {
		close(c.closeChan)
	}
	c.basicClient.close(reason)
}

func (c *passiveClient) Close() {
	c.close(false)
}
