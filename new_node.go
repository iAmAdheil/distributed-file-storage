package main

import (
	"log"

	"github.com/iAmAdheil/distributed-file-storage/db"
	"github.com/iAmAdheil/distributed-file-storage/http_server"
	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

// Each node has 2 servers running -> an entry point http server, an internal file server

type Node struct {
	fileServer *FileServer
	httpServer *http_server.HTTPServer
}

func configNode(listenaddr string, httpaddr string, nodes []string) *Node {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddress: listenaddr,
		Handshake:     p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	storeOpts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
		Root:              listenaddr + "_network",
	}
	dbOpts := db.DBOpts{
		Filename: listenaddr + ".db",
	}

	db := db.NewDB(dbOpts)

	// pass tcp transport here
	// make it generic, by passing the to be used transport layer from outside
	// instead of locking it from inside server
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	fileServerOpts := FileServerOpts{
		EncKey:    newEncryptionKey(),
		transport: tcpTransport,

		StoreOpts: storeOpts,

		BootStrapNodes: nodes,
		ID:             genID(),
	}

	fileServer := NewFileServer(fileServerOpts)
	tcpTransport.OnPeer = fileServer.OnPeer

	httpServerOpts := http_server.HTTPServerOpts{
		HttpAddr: httpaddr,
		Internal: fileServer,
		DB:       db,
	}

	httpServer := http_server.NewHTTPServer(httpServerOpts)

	return &Node{
		fileServer: fileServer,
		httpServer: httpServer,
	}
}

func (n *Node) StartNode() error {
	log.Printf("Starting node with http addr (%s)\n", n.httpServer.HttpAddr)
	if err := n.fileServer.StartFileServer(); err != nil {
		return err
	}
	if err := n.httpServer.StartHttpServer(); err != nil {
		return err
	}
	return nil
}
