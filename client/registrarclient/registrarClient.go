package registrarclient

import (
	"context"
	"fmt"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"time"
)

const (
	HEARTBEATOFFSET  = 1
	MAXRETRY         = 2
	DEFAULTLEASETIME = 10
	BufSize          = 10
)

type RegistrarClient struct {
	options   *ClientOpts
	addrIndex int
	uniqueID  string

	cli    pb.EtcdRegistrarClient
	conn   *grpc.ClientConn
	ticker *time.Ticker
	close  chan struct{}
}

func NewRegistrarClient(option ...ClientOption) *RegistrarClient {
	opts := NewDefaultOptions()
	opts.ApplyOpts(option)

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
	c.Close()
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
			return nil
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
	c.uniqueID = resp.GetServiceName()
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
				Name:    c.uniqueID,
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
				log.Println(c.uniqueID, c.options.localAddr, "heartbeat success")
				retryTimes = 0
			}
		case <-c.close:
			log.Println(c.uniqueID, "close time tick")
			return
		}
	}
}

func (c *RegistrarClient) logout(ctx context.Context) error {
	_, err := c.cli.Logout(ctx, &pb.Service{
		Name:    c.uniqueID,
		Address: c.options.localAddr,
	})
	return err
}

// TODO new了但没注册
func (c *RegistrarClient) Close() {
	if c.ticker != nil {
		close(c.close)
		c.ticker.Stop()
		c.ticker = nil
	}
	if c.cli != nil {
		_ = c.logout(context.Background())
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

func (c *RegistrarClient) Subscribe(ctx context.Context, name string) (chan *pb.SubscribeResponse, error) {
	stream, err := c.cli.Subscribe(ctx, &pb.SubscribeRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}
	ch := make(chan *pb.SubscribeResponse, BufSize)
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					log.Println("stream end.")
					close(ch)
				}
				return
			}
			ch <- resp
		}
	}()
	return ch, nil
}
