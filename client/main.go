package main

import (
	"context"
	"etcd-registrar/client/registrarclient"
	"log"
)

func main() {
	cli, err := registrarclient.NewRegistrarClient("localhost:9090")
	if err != nil {
		log.Fatalln(cli)
	}
	defer cli.Close()
	err = cli.Register(context.Background(), "test", "test_address", 10)
	if err != nil {
		log.Fatalln(cli)
	}
	select {}

}
