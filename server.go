package main

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"sync"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

type Message struct {
	From    string
	Payload any
}

func (fServer *FileServer) broadcast(m *Message) error {
	peers := []io.Writer{}
	for _, peer := range fServer.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(*m)
}

func (fServer *FileServer) StoreData(key string, r io.Reader) error {
	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

	buf := new(bytes.Buffer)
	msg := Message{
		Payload: []byte("storagekey"),
	}
	gob.NewEncoder(buf).Encode(msg)

	for _, peer := range fServer.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	time.Sleep(time.Second * 3)

	data := []byte("THIS IS A LARGE FILE")
	for _, peer := range fServer.peers {
		if err := peer.Send(data); err != nil {
			return err
		}
	}

	return nil

	// if err := fServer.store.Write(key, tee); err != nil {
	// 	return err
	// }

	// p := DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }

	// return fServer.broadcast(&Message{
	// 	From:    "todo",
	// 	Payload: p,
	// })
}

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

func (fServer *FileServer) loop() {
	defer func() {
		log.Println("server stopped by user quit action")
		if err := fServer.transport.Close(); err != nil {
			log.Println("TCP transport closing error:", err)
		}
	}()

	for {
		select {
		case rpc := <-fServer.transport.Consume():
			var m *Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&m); err != nil {
				log.Println("error when decoding message: ", err)
			}
			log.Println("received message: ", string(m.Payload.([]byte)))

			buf := make([]byte, 1000)
			peer := fServer.peers[rpc.From]
			if _, err := peer.Read(buf); err != nil {
				log.Println("error when reading large data: ", err)
			}

			log.Println("received message: ", string(string(buf)))
			peer.(*p2p.TCPPeer).Wg.Done()

		case <-fServer.quitch:
			return
		}
	}
}

// func handleMessage(rpc *p2p.RPC) {
// 	var m *Message
// 	if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&m); err != nil {
// 		log.Println("error when decoding message: ", err)
// 	}
// 	log.Println("received message: ", string(m.Payload.([]byte)))
// }

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

func (fServer *FileServer) OnPeer(p p2p.Peer) error {
	fServer.peerLock.Lock()
	defer fServer.peerLock.Unlock()

	fServer.peers[p.RemoteAddr().String()] = p
	log.Println("remote peer added: ", p.RemoteAddr().String())

	return nil
}
