package registrarclient

import (
	"context"
	"errors"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"github.com/ChenaLi0816/etcd-registrar/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"sync/atomic"
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
	// TODO 多次switch发生冲突，dial和close同时
	options   *ClientOpts
	addrIndex int
	uniqueID  string

	cli      pb.EtcdRegistrarClient
	conn     *grpc.ClientConn
	isClosed atomic.Bool
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
	if c.uniqueID == "" {
		return nil
	}
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
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Println("conn close err:", err)
		}
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
		Name:      name,
		LocalAddr: c.options.localAddr,
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
				addrIndex: int(utils.RandInt(int64(len(opts.address)), false)),
				uniqueID:  "",
				cli:       nil,
				conn:      nil,
				isClosed:  atomic.Bool{},
			},
			closeChan: nil,
		}
		c.basicClient.isClosed.Store(false)
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
				addrIndex: int(utils.RandInt(int64(len(opts.address)), false)),
				uniqueID:  "",
				cli:       nil,
				conn:      nil,
				isClosed:  atomic.Bool{},
			},
			ticker:    nil,
			closeChan: nil,
		}
		c.isClosed.Store(false)
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
