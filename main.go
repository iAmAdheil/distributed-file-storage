package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

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
		EncKey:         newEncryptionKey(),
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
			log.Fatal("server start failed: ", err)
		}
	}()

	time.Sleep(1 * time.Second)

	go func() {
		if err := s2.Start(); err != nil {
			log.Fatal("server start failed: ", err)
		}
	}()

	time.Sleep(1 * time.Second)

	// multi write test
	// data := []byte("this is my private data")
	// for i := 0; i < 10; i++ {
	// 	s2.Store(fmt.Sprintf("myprivatedata_%v", i), bytes.NewReader(data))
	// }

	// single write test
	key := "myprivatedata"
	data := []byte("this is my private data")
	s2.Store(key, bytes.NewReader(data))

	if err := s2.store.Delete(key); err != nil {
		log.Fatal("delete failed: ", err)
	}

	// read test
	buf := make([]byte, 1028)
	r, err := s2.Get(key)
	if err != nil {
		log.Fatal("get data failed: ", err)
	}
	if _, err := r.Read(buf); err != nil {
		log.Fatal("read failed: ", err)
	}
	fmt.Println("Data received: ", string(buf))

	select {}
}
