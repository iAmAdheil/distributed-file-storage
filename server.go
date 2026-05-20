package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

type Message struct {
	Payload any
}

func (fServer *FileServer) stream(m *Message) error {
	peers := []io.Writer{}
	for _, peer := range fServer.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(*m)
}

func (fServer *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Println("err whend decoding message: ", err)
		return err
	}

	for _, peer := range fServer.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

type MessageGetFile struct {
	Key string
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

// Currently check if file exists locally, if not stream it from another peer
// Ideally ask peers with the stored file, then choose one peer to stream it
// and then listen for data stream from that peer
func (fServer *FileServer) Get(key string) (io.Reader, error) {
	ok := fServer.store.Has(key)
	if !ok {
		r, err := fServer.store.Read(key)
		if err != nil {
			return r, err
		}
	}

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fServer.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Second * 3)

	for _, peer := range fServer.peers {
		fileBuffer := new(bytes.Buffer)
		_, err := io.Copy(fileBuffer, peer)
		if err != nil {
			// return nil, err
			continue
		}
		return fileBuffer, nil
	}

	return nil, nil
}

func (fServer *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuf = new(bytes.Buffer)
		tee     = io.TeeReader(r, fileBuf)
	)

	size, err := fServer.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err := fServer.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Second * 3)

	for _, peer := range fServer.peers {
		if err := peer.Send(fileBuf.Bytes()); err != nil {
			return err
		}
	}

	return nil
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

			if err := fServer.handleMessage(rpc.From, m); err != nil {
				log.Println("some error during message handling: ", err)
				return
			}

		case <-fServer.quitch:
			return
		}
	}
}

func (fServer *FileServer) handleMessage(from string, m *Message) error {
	switch payload := m.Payload.(type) {
	case MessageStoreFile:
		if err := fServer.handleMessageStoreFile(from, &payload); err != nil {
			log.Println("error when storing streamed data: ", err)
		}
	case MessageGetFile:
		if err := fServer.handleMessageGetFile(from, &payload); err != nil {
			log.Println("error when streaming stored data: ", err)
		}
	}

	return nil
}

func (fServer *FileServer) handleMessageGetFile(from string, m *MessageGetFile) error {
	r, err := fServer.store.Read(m.Key)
	if err != nil {
		return err
	}

	peer, ok := fServer.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not found in peer map", from)
	}

	_, err = io.Copy(peer, r)
	if err != nil {
		return err
	}

	return nil
}

func (fServer *FileServer) handleMessageStoreFile(from string, m *MessageStoreFile) error {
	peer, ok := fServer.peers[from]
	defer peer.(*p2p.TCPPeer).Wg.Done()

	if !ok {
		return fmt.Errorf("peer [%s] not found in server peer map", from)
	}

	if _, err := fServer.store.Write(m.Key, io.LimitReader(peer, m.Size)); err != nil {
		return err
	}

	return nil
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

func (fServer *FileServer) OnPeer(p p2p.Peer) error {
	fServer.peerLock.Lock()
	defer fServer.peerLock.Unlock()

	fServer.peers[p.RemoteAddr().String()] = p
	log.Println("remote peer added: ", p.RemoteAddr().String())

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
