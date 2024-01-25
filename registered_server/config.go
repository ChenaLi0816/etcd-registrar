package registered_server

import (
	"github.com/ChenaLi0816/etcd-registrar/client/registrarclient"
	"google.golang.org/grpc"
)

type registerFunc func(grpc.ServiceRegistrar)

type GrpcServerConfig struct {
	Network      string
	Address      string
	RegisterOpt  *registrarclient.ClientOpts
	RegisterFunc registerFunc
}
