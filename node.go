package main

import (
	"fmt"

	"github.com/iAmAdheil/distributed-file-storage/file_server"
	"github.com/iAmAdheil/distributed-file-storage/http_server"
)

// Each node has 2 servers running -> an entry point http server, an internal file server

type Server interface {
	Start() error
	Address() string
}

type Node struct {
	servers []Server
}

func configNode(serveraddr string, httpaddr string, nodes []string) *Node {
	var servers []Server

	fileServerOpts := file_server.FileServerOpts{
		ListenAddress:  serveraddr,
		BootStrapNodes: nodes,
		ID:             genID(),
	}
	fileServer := file_server.New(fileServerOpts)

	httpServerOpts := http_server.HTTPServerOpts{
		HttpAddr: httpaddr,
		Internal: fileServer,
	}
	httpServer := http_server.New(httpServerOpts)

	servers = append(servers, fileServer, httpServer)

	return &Node{
		servers: servers,
	}
}

func (n *Node) Start() {
	for _, v := range n.servers {
		if err := v.Start(); err != nil {
			fmt.Printf("Error when starting server with listen addr (%s): %s\n", v.Address(), err.Error())
		}
	}
}
