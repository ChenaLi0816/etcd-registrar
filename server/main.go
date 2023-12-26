package main

import (
	"etcd-registrar/proto/pb"
	"etcd-registrar/server/registrarserver"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	lis, _ := net.Listen("tcp", ":9090")
	grpcServer := grpc.NewServer()
	pb.RegisterEtcdRegistrarServer(grpcServer, registrarserver.NewEtcdRegistrarServer("localhost:2379"))
	log.Println("server prepared.")
	_ = grpcServer.Serve(lis)
}
