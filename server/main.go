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
	lisPort, etcdPort := initFlag()
	//lisPort, etcdPort := "192.168.1.7:8080", ":2379"
	lisAddr := "127.0.0.1" + lisPort
	lis, _ := net.Listen("tcp", lisAddr)
	grpcServer := grpc.NewServer()
	pb.RegisterEtcdRegistrarServer(grpcServer, registrarserver.NewEtcdRegistrarServer(lisAddr, "127.0.0.1"+etcdPort, registrarserver.RoundRobin))
	log.Println("server prepared on", lisPort)
	_ = grpcServer.Serve(lis)
}

func initFlag() (string, string) {
	if len(os.Args) < 3 {
		log.Fatalln("need flag")
	}
	return ":" + os.Args[1], ":" + os.Args[2]
}
