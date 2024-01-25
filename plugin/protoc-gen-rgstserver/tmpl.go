package main

const tmplString = `package main

import (
	"github.com/ChenaLi0816/etcd-registrar/client/registrarclient"
	"github.com/ChenaLi0816/etcd-registrar/registered_server"
    "google.golang.org/grpc"
	"{{.PbPath}}"
	"{{.ServerpbPath}}"
)

const (
	NAME = "your_service_name"
	ADDR = "your_service_addr"
)
{{$PbName := .PbName}}{{$ServerpbName := .ServerpbName}}
func main() {
	cfg := &registered_server.GrpcServerConfig{
		Network:     "tcp",
		Address:     ADDR,
		RegisterOpt: registrarclient.NewDefaultOptions().WithService(NAME, ADDR).WithRegistrarAddress([]string{"your_registrar_addr"}),
		RegisterFunc: func(grpcServer grpc.ServiceRegistrar) {
		    {{range .Service}}{{$PbName}}.Register{{.ServiceName}}Server(grpcServer, {{$ServerpbName}}.New{{.ServiceName}}Server())
		    {{end}}},
	}

	s := registered_server.NewServer(cfg)
	if err := s.Run(); err != nil {
		panic(err)
	}
}`
