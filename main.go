package main

import (
	"fmt"
	"log"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

func OnPeerFunc(peer p2p.Peer) error {
	fmt.Println("work with peer from outside TCP transport here")
	return nil
}

func main() {
	s1 := configNode(":3000", ":8001", []string{})
	s2 := configNode(":4000", ":8002", []string{":3000"})
	s3 := configNode(":7000", ":8003", []string{":3000", ":4000"})

	if err := s1.StartNode(); err != nil {
		log.Fatal("server start failed: ", err)
	}

	time.Sleep(1 * time.Second)

	go func() {
		if err := s2.StartNode(); err != nil {
			log.Fatal("server start failed: ", err)
		}
	}()

	time.Sleep(1 * time.Second)

	go func() {
		if err := s3.StartNode(); err != nil {
			log.Fatal("server start failed: ", err)
		}
	}()

	time.Sleep(1 * time.Second)

	select {}
}
