package registrarclient

import (
	"context"
	"etcd-registrar/proto/pb"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

const (
	HEARTBEATOFFSET  = 1
	MAXRETRY         = 2
	DEFAULTLEASETIME = 10
)

type RegistrarClient struct {
	options   *ClientOpts
	addrIndex int

	cli    pb.EtcdRegistrarClient
	conn   *grpc.ClientConn
	ticker *time.Ticker
	close  chan struct{}
}

func NewRegistrarClient(option ...ClientOption) *RegistrarClient {
	opts := NewDefaultOptions()
	opts.ApplyOpts(option)

	// TODO delete
	name := opts.GetName()
	addr := opts.GetLocalAddr()
	leaseTime := opts.GetLeaseTime()
	log.Println(name, addr, "create with lease time", leaseTime)

	c := &RegistrarClient{
		options:   opts,
		addrIndex: 0,
		cli:       nil,
		conn:      nil,
		ticker:    nil,
		close:     nil,
	}
	_ = c.switchConnection(context.Background())

	return c
}

func (c *RegistrarClient) switchConnection(ctx context.Context) error {
	c.Close(ctx)
	l := len(c.options.address)
	oft := 0
	for oft < l {
		c.addrIndex++
		oft++
		if c.addrIndex >= l {
			c.addrIndex = 0
		}
		a := c.options.address[c.addrIndex]
		var err error
		if c.options.passive {
			//TODO passive
		} else {
			err = c.newGrpcConn(a)
		}

		if err != nil {
			log.Println("warning:registrar address", a, "is unavailable")
		} else {
			log.Println("connect to registrar", a, "success")
			break
		}
	}

	if oft == len(c.options.address) {
		panic("no server address is available")
	}
	return nil
}

func (c *RegistrarClient) newGrpcConn(addr string) error {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	cli := pb.NewEtcdRegistrarClient(conn)
	c.cli = cli
	c.conn = conn
	c.ticker = nil
	c.close = make(chan struct{}, 1)
	return nil
}

func (c *RegistrarClient) Register(ctx context.Context) error {
	if c.ticker != nil {
		return fmt.Errorf("service time ticker is not nil")
	}
	req := &pb.RegisterRequest{
		Name:      c.options.name,
		Address:   c.options.localAddr,
		LeaseTime: c.options.leaseTime,
	}
	resp, err := c.cli.Register(ctx, req)
	if err != nil {
		log.Println("register err:", err, "now retry")
		_ = c.switchConnection(ctx)
	}
	c.options.name = resp.GetServiceName()
	c.ticker = time.NewTicker(time.Second * time.Duration(c.options.leaseTime-HEARTBEATOFFSET))
	log.Println("register success")
	go c.timeTick(ctx)
	return nil
}

func (c *RegistrarClient) timeTick(ctx context.Context) {
	retryTimes := 0
	for {
		select {
		case <-c.ticker.C:
			_, err := c.cli.HeartbeatActive(ctx, &pb.Service{
				Name:    c.options.name,
				Address: c.options.localAddr,
			})
			if err != nil {
				log.Println("heartbeat err:", err)
				retryTimes++
				if retryTimes >= MAXRETRY {
					log.Println("heartbeat err overtimes, switch connection")
					_ = c.switchConnection(ctx)
					err := c.Register(ctx)
					if err != nil {
						log.Println("retry register err:", err)
					}
					return
				}
			} else {
				log.Println(c.options.name, c.options.localAddr, "heartbeat success")
				retryTimes = 0
			}
		case <-c.close:
			log.Println(c.options.name, "close time tick")
			return
		}
	}
}

func (c *RegistrarClient) logout(ctx context.Context) error {
	_, err := c.cli.Logout(ctx, &pb.Service{
		Name:    c.options.name,
		Address: c.options.localAddr,
	})
	return err
}

// TODO new了但没注册
func (c *RegistrarClient) Close(ctx context.Context) {
	if c.ticker != nil {
		close(c.close)
		c.ticker.Stop()
		c.ticker = nil
	}
	if c.cli != nil {
		_ = c.logout(ctx)
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}

}

func (c *RegistrarClient) Discover(ctx context.Context, name string) (string, error) {
	resp, err := c.cli.Discover(ctx, &pb.DiscoverRequest{
		Name: name,
	})
	if err != nil {
		return "", err
	}
	return resp.GetAddress(), nil
}
