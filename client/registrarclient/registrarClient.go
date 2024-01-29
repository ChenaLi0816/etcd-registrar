package registrarclient

import (
	"context"
	"errors"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"time"
)

const (
	HEARTBEATOFFSET  = 1
	DEFAULTLEASETIME = 10
	BufSize          = 10
)

var (
	errServerUnavailable = errors.New("no server address is available")
)

type RegistrarClient interface {
	Register(ctx context.Context) error
	Close()
	Discover(ctx context.Context, name string) (string, error)
	Subscribe(ctx context.Context, name string) (chan *pb.SubscribeResponse, error)
}

type basicClient struct {
	options   *ClientOpts
	addrIndex int
	uniqueID  string

	cli  pb.EtcdRegistrarClient
	conn *grpc.ClientConn
}

func (c *basicClient) newGrpcConn(addr string) error {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	cli := pb.NewEtcdRegistrarClient(conn)
	c.cli = cli
	c.conn = conn

	return nil
}

func (c *basicClient) logout(ctx context.Context) error {
	_, err := c.cli.Logout(ctx, &pb.Service{
		Name:    c.uniqueID,
		Address: c.options.localAddr,
	})
	return err
}

// reason - whether the close is because client error
func (c *basicClient) close(reason bool) {
	if c.cli != nil && !reason {
		if err := c.logout(context.Background()); err != nil {
			log.Println("logout err:", err)
		}
		c.cli = nil
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Println("conn close err:", err)
		}
		c.conn = nil
	}
}

func (c *basicClient) Discover(ctx context.Context, name string) (string, error) {
	resp, err := c.cli.Discover(ctx, &pb.DiscoverRequest{
		Name: name,
	})
	if err != nil {
		return "", err
	}
	return resp.GetAddress(), nil
}

func (c *basicClient) Subscribe(ctx context.Context, name string) (chan *pb.SubscribeResponse, error) {
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

func NewRegistrarClient(opts *ClientOpts) RegistrarClient {
	//opts := NewDefaultOptions()
	//opts.ApplyOpts(option)
	if opts == nil {
		opts = NewDefaultOptions()
	}

	if opts.passive {
		c := &passiveClient{
			basicClient: basicClient{
				options:   opts,
				addrIndex: 0,
				uniqueID:  "",
				cli:       nil,
				conn:      nil,
			},
			closeChan: nil,
		}
		err := c.switchConnection(context.Background())
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		return c
	} else {
		c := &activeClient{
			basicClient: basicClient{
				options:   opts,
				addrIndex: 0,
				uniqueID:  "",
				cli:       nil,
				conn:      nil,
			},
			ticker:    nil,
			closeChan: nil,
		}
		err := c.switchConnection(context.Background())
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		return c
	}
}

func (c *basicClient) switchConnection(ctx context.Context) error {
	c.close(true)
	l := len(c.options.address)
	oft := 0
	for oft < l {
		c.addrIndex++
		oft++
		if c.addrIndex >= l {
			c.addrIndex = 0
		}
		a := c.options.address[c.addrIndex]
		err := c.newGrpcConn(a)

		if err != nil {
			log.Println("warning:registrar address", a, "is unavailable")
		} else {
			log.Println("connect to registrar", a, "success")
			return nil
		}
	}

	return errServerUnavailable
}
