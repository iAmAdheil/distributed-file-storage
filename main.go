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

func configServer(listenaddr string, httpaddr string, nodes []string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddress: listenaddr,
		Handshake:     p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	storeOpts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
		Root:              listenaddr + "_network",
	}
	httpServerOpts := HttpOpts{
		HttpAddr: httpaddr,
	}

	// pass tcp transport here
	// make it generic, by passing the to be used transport layer from outside
	// instead of locking it from inside server
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	fileServerOpts := FileServerOpts{
		EncKey:    newEncryptionKey(),
		transport: tcpTransport,

		StoreOpts: storeOpts,
		HttpOpts:  httpServerOpts,

		BootStrapNodes: nodes,
		ID:             genID(),
	}

	server := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = server.OnPeer

	return server
}

func main() {
	s1 := configServer(":3000", ":8001", []string{})
	s2 := configServer(":4000", ":8002", []string{":3000"})
	s3 := configServer(":7000", ":8003", []string{":3000", ":4000"})

	if err := s1.Start(); err != nil {
		log.Fatal("server start failed: ", err)
	}

	time.Sleep(1 * time.Second)

	go func() {
		if err := s2.Start(); err != nil {
			log.Fatal("server start failed: ", err)
		}
	}()

	time.Sleep(1 * time.Second)

	go func() {
		if err := s3.Start(); err != nil {
			log.Fatal("server start failed: ", err)
		}
	}()

	time.Sleep(1 * time.Second)

	select {}
}
