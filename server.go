package main

import (
	"log"
	"sync"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

type FileServerOpts struct {
	StoreOpts
	transport p2p.Transport

	BootStrapNodes []string
}

type FileServer struct {
	FileServerOpts

	store *Store

	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	quitch   chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	server := &FileServer{
		FileServerOpts: opts,

		store: NewStore(opts.StoreOpts),

		peerLock: sync.Mutex{},
		peers:    make(map[string]p2p.Peer),
		quitch:   make(chan struct{}),
	}

	return server
}

func (fServer *FileServer) Start() error {
	if err := fServer.transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(fServer.BootStrapNodes) > 0 {
		fServer.bootstrapNodes()
	}
	fServer.loop()

	return nil
}

func (fServer *FileServer) OnPeer(p p2p.Peer) error {
	fServer.peerLock.Lock()
	defer fServer.peerLock.Unlock()

	fServer.peers[p.RemoteAddr().String()] = p
	log.Println("remote peer added: ", p.RemoteAddr().String())

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

func (fServer *FileServer) bootstrapNodes() {
	for _, addr := range fServer.BootStrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(address string) {
			if err := fServer.transport.Dial(address); err != nil {
				log.Printf("error when dialing a remote node %s:, %s\n", address, err)
			}
		}(addr)
	}
}

func (fServer *FileServer) Stop() {
	close(fServer.quitch)
}
