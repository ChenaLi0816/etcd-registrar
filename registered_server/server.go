package registered_server

import (
	"context"
	"errors"
	"github.com/ChenaLi0816/etcd-registrar/client/registrarclient"
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"google.golang.org/grpc"
	"net"
)

type Server interface {
	Run() error
	Stop()
	Discover(ctx context.Context, name string) (string, error)
	Subscribe(ctx context.Context, name string) (chan *pb.SubscribeResponse, error)
}

type server struct {
	isRegistered bool
	registrarclient.RegistrarClient

	grpcServer *grpc.Server
	lis        net.Listener
}

func (s *server) Run() error {
	err := s.Register(context.Background())
	if err != nil {
		return err
	}
	return s.grpcServer.Serve(s.lis)
}

func (s *server) Stop() {
	s.grpcServer.Stop()
	s.Close()
}

func NewServer(cfg *GrpcServerConfig, opt ...grpc.ServerOption) Server {
	lis, err := net.Listen(cfg.Network, cfg.Address)
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer(opt...)
	//mypb.RegisterEtcdRegistrarServer(grpcServer, mypb_server.NewEtcdRegistrarServer())
	if cfg.RegisterFunc == nil {
		panic(errors.New("register func is nil"))
	}
	cfg.RegisterFunc(grpcServer)

	s := &server{
		isRegistered:    false,
		RegistrarClient: registrarclient.NewRegistrarClient(cfg.RegisterOpt),
		grpcServer:      grpcServer,
		lis:             lis,
	}
	return s
}
