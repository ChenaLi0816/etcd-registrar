package main

import (
	"etcd-registrar/proto/pb"
	"etcd-registrar/server/registrarserver"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	//port := ":" + initFlag()
	port := ":8080"
	lis, _ := net.Listen("tcp", port)
	grpcServer := grpc.NewServer()
	pb.RegisterEtcdRegistrarServer(grpcServer, registrarserver.NewEtcdRegistrarServer("localhost:2379"))
	log.Println("server prepared on", port)
	_ = grpcServer.Serve(lis)
}

func initFlag() string {
	if len(os.Args) < 2 {
		log.Fatalln("need flag")
	}
	return os.Args[1]
}
