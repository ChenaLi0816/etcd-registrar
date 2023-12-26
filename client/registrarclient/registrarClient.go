package registrarclient

import (
	"context"
	"etcd-registrar/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

const (
	HEARTBEATOFFSET = 1
	MAXRETRY        = 2
)

type RegistrarClient struct {
	cli     pb.EtcdRegistrarClient
	conn    *grpc.ClientConn
	name    string
	address string
	ticker  *time.Ticker
	close   chan struct{}
}

func NewRegistrarClient(addr string) (*RegistrarClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	cli := pb.NewEtcdRegistrarClient(conn)
	return &RegistrarClient{
		cli:    cli,
		conn:   conn,
		name:   "",
		ticker: nil,
		close:  make(chan struct{}, 1),
	}, nil
}

func (c *RegistrarClient) Register(ctx context.Context, name, addr string, leaseTime int64) error {
	req := &pb.RegisterRequest{
		Name:      name,
		Address:   addr,
		LeaseTime: leaseTime,
	}
	resp, err := c.cli.Register(ctx, req)
	if err != nil {
		return err
	}
	c.name = resp.GetServiceName()
	c.address = addr
	if c.ticker == nil {
		c.ticker = time.NewTicker(time.Second * time.Duration(leaseTime-HEARTBEATOFFSET))
		go c.timeTick(ctx)
	}

	return nil
}

func (c *RegistrarClient) timeTick(ctx context.Context) {
	retryTimes := 0
	for {
		select {
		case <-c.ticker.C:
			_, err := c.cli.HeartbeatActive(ctx, &pb.Service{
				Name:    c.name,
				Address: c.address,
			})
			if err != nil {
				log.Println("heartbeat err:", err)
				retryTimes++
				if retryTimes >= MAXRETRY {
					c.Close()
					return
				}
			} else {
				log.Println(c.name, c.address, "heartbeat success")
				retryTimes = 0
			}
		case <-c.close:
			log.Println(c.name, "close time tick")
			return
		}
	}
}

func (c *RegistrarClient) logout(ctx context.Context) error {
	_, err := c.cli.Logout(ctx, &pb.Service{
		Name:    c.name,
		Address: c.address,
	})
	return err
}

func (c *RegistrarClient) Close() {
	select {
	case c.close <- struct{}{}:
		break
	default:
		break
	}

	c.ticker.Stop()
	_ = c.logout(context.Background())
	_ = c.conn.Close()
}
