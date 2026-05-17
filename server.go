package main

import (
	"log"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

type FileServerOpts struct {
	p2p.TCPTransportOpts
	StoreOpts
}

type FileServer struct {
	FileServerOpts

	store     *Store
	transport p2p.Transport
	quitch    chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	return &FileServer{
		FileServerOpts: opts,
		transport:      p2p.NewTCPTransport(opts.TCPTransportOpts),
		store:          NewStore(opts.StoreOpts),
		quitch:         make(chan struct{}),
	}
}

func (fServer *FileServer) Start() error {
	if err := fServer.transport.ListenAndAccept(); err != nil {
		return err
	}

	fServer.loop()

	return nil
}

func (fServer *FileServer) loop() {
	defer func() {
		log.Println("server stopped by user quit action")
		if err := fServer.transport.Close(); err != nil {
			log.Println("TCP transport closing error:", err)
		}
	}()

	for {
		select {
		case msg := <-fServer.transport.Consume():
			log.Println("message:", msg)
		case <-fServer.quitch:
			return
		}
	}
}

func (fServer *FileServer) Stop() {
	close(fServer.quitch)
}
