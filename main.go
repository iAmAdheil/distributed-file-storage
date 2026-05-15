package main

import (
	"fmt"
	"log"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

func OnPeerFunc(peer p2p.Peer) error {
	fmt.Println("work with peer from outside TCP transport here")
	return nil
}

func main() {
	opts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		Handshake:     p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeerFunc,
	}
	tr := p2p.NewTCPTransport(opts)

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal("some tcp error ahhhh:", err)
	}

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("received message: %v\n", msg)
		}
	}()

	select {}
}
