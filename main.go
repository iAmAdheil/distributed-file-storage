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
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		Handshake:     p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// OnPeer:        OnPeerFunc,
	}
	storeOpts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	fileServerOpts := FileServerOpts{
		TCPTransportOpts: tcpTransportOpts,
		StoreOpts:        storeOpts,
	}

	server := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(3 * time.Second)
		server.Stop()
	}()

	if err := server.Start(); err != nil {
		log.Fatal("server start failed:", err)
	}
}
