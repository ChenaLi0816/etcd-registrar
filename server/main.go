package main

import (
	"github.com/ChenaLi0816/etcd-registrar/proto/pb"
	"github.com/ChenaLi0816/etcd-registrar/server/registrarserver"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	//lisPort, etcdPort := initFlag()
	lisPort, _ := "8080", "2379"
	lisAddr := "127.0.0.1:" + lisPort
	etcdAddr := []string{"127.0.0.1:4379", "127.0.0.1:5379", "127.0.0.1:7379"}
	//etcdAddr := "127.0.0.1:" + etcdPort
	lis, _ := net.Listen("tcp", lisAddr)
	grpcServer := grpc.NewServer()
	pb.RegisterEtcdRegistrarServer(grpcServer, registrarserver.NewEtcdRegistrarServer(lisAddr, etcdAddr, registrarserver.WeightRoundRobin))
	log.Println("server prepared on", lisPort)
	_ = grpcServer.Serve(lis)
}

func initFlag() (string, string) {
	if len(os.Args) < 3 {
		log.Fatalln("need flag")
	}
	return os.Args[1], os.Args[2]
}
