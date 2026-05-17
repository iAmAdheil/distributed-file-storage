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

func configServer(listenaddr string, nodes []string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddress: listenaddr,
		Handshake:     p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	storeOpts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
		Root:              listenaddr + "_network",
	}

	// pass tcp transport here
	// make it generic, by passing the to be used transport layer from outside
	// instead of locking it from inside server
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		transport:      tcpTransport,
		StoreOpts:      storeOpts,
		BootStrapNodes: nodes,
	}

	server := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = server.OnPeer

	return server
}

func main() {
	s1 := configServer(":3000", []string{})
	s2 := configServer(":4000", []string{":3000"})

	go func() {
		if err := s1.Start(); err != nil {
			log.Fatal("server start failed:", err)
		}
	}()

	if err := s2.Start(); err != nil {
		log.Fatal("server start failed:", err)
	}
}
