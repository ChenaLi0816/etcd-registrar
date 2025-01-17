package registrarclient

import (
	"context"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"log"
	"os"
	"strings"
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
		panic("service name or service address is null")
	}
	req := &pb.RegisterRequest{
		Name:      c.options.name,
		Address:   c.options.localAddr,
		LeaseTime: c.options.leaseTime,
		Weight:    c.options.weight,
		Version:   c.options.version,
	}
	resp, err := c.cli.Register(ctx, req)
	if err != nil {
		log.Println("register err:", err, "now retry")
		err = c.switchConnection(ctx)
		if err != nil {
			if err == errCliIsClosing {
				return err
			}
			log.Println(err)
			os.Exit(1)
		}
		resp, err = c.cli.Register(ctx, req)
	}
	if err != nil {
		return err
	}
	c.uniqueID = resp.GetServiceName()
	if resp.RegistrarAddr != nil {
		log.Println("get addr:", resp.RegistrarAddr)
		c.options.address = resp.RegistrarAddr
	}

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
		select {
		case <-c.closeChan:
			log.Println(c.uniqueID, "stop heartbeat")
		default:
			log.Println("switch connection")
			err = c.switchConnection(ctx)
			if err != nil {
				if err == errCliIsClosing {
					return
				}
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
			resp, err := stream.Recv()
			if err != nil {
				log.Println("stream recv err:", err)
				return
			}
			rawInfo := resp.GetInfo()
			if rawInfo != "" {
				log.Println("get info:", rawInfo)
				for _, info := range strings.Split(rawInfo, ";") {
					i := strings.Index(info, ":")
					t, msg := info[:i], info[i+1:]
					switch t {
					case "registrar":
						c.options.address = strings.Split(msg, ",")
					case "version":
						if c.options.version != msg {
							log.Printf("version %s confirmed, version %s exit", msg, c.options.version)
							c.Close()
							return
						}
					}
				}
			}
			if err = stream.Send(&pb.CheckHealth{}); err != nil {
				log.Println("stream send err:", err)
				return
			}
			log.Println("heartbeat passive success", c.uniqueID)
		}

	}
}

func (c *passiveClient) close() {
	if c.closeChan != nil {
		close(c.closeChan)
	}
	c.basicClient.close(false)

}

func (c *passiveClient) Close() {
	if c.isClosed.Swap(true) {
		return
	}
	c.close()
}
