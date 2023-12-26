package main

import (
	"context"
	"etcd-registrar/client/registrarclient"
	"log"
	"time"
)

func main() {
	cli1, err := registrarclient.NewRegistrarClient("localhost:9090")
	if err != nil {
		log.Fatalln(err)
	}
	defer cli1.Close()
	err = cli1.Register(context.Background(), "test", "test_address1", 10)

	cli2, err := registrarclient.NewRegistrarClient("localhost:9090")
	if err != nil {
		log.Fatalln(err)
	}
	defer cli2.Close()
	err = cli2.Register(context.Background(), "test", "test_address2", 15)

	cli3, err := registrarclient.NewRegistrarClient("localhost:9090")
	if err != nil {
		log.Fatalln(err)
	}
	defer cli3.Close()
	err = cli3.Register(context.Background(), "test", "test_address3", 5)
	if err != nil {
		log.Fatalln(err)
	}

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	name := "test"
	for {
		select {
		case <-ticker.C:
			addr, err := cli1.Discover(context.Background(), name)
			if err != nil {
				log.Fatalln(err)
			}
			log.Println("get service", name, "in", addr)
		}
	}

}
