package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

type Message struct {
	ID      string
	Payload any
}

// func (fServer *FileServer) stream(m *Message) error {
// 	peers := []io.Writer{}
// 	for _, peer := range fServer.peers {
// 		peers = append(peers, peer)
// 	}
// 	mw := io.MultiWriter(peers...)
// 	return gob.NewEncoder(mw).Encode(*m)
// }

func (fServer *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Printf("[%s] err when decoding message before broadcast: %s\n", fServer.transport.Addr(), err)
		return err
	}

	for _, peer := range fServer.peers {
		if err := peer.Send([]byte{p2p.IncomingMessageByte}); err != nil {
			return err
		}
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
		// fmt.Printf("[%s] message broadcasted to: [%s]", fServer.transport.Addr(), peer.RemoteAddr())
	}

	return nil
}

type MessageGetFile struct {
	Key string
}

type MessageDeleteFile struct {
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
	ok := fServer.store.Has(key, fServer.ID)
	if ok {
		_, r, err := fServer.store.Read(key, fServer.ID)
		if err == nil {
			return r, nil
		}
	}

	msg := Message{
		ID: fServer.ID,
		Payload: MessageGetFile{
			Key: hashKey(key),
		},
	}
	if err := fServer.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(5 * time.Millisecond)

	for _, peer := range fServer.peers {
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		_, err := fServer.store.DWrite(fServer.EncKey, key, fServer.ID, io.LimitReader(peer, fileSize))
		if err != nil {
			// return nil, err
			continue
		}

		peer.CloseStream()
	}

	_, r, err := fServer.store.Read(key, fServer.ID)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (fServer *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuf = new(bytes.Buffer)
		tee     = io.TeeReader(r, fileBuf)
	)

	size, err := fServer.store.Write(key, fServer.ID, tee)
	if err != nil {
		return err
	}

	msg := Message{
		ID: fServer.ID,
		Payload: MessageStoreFile{
			Key:  hashKey(key),
			Size: size + 16,
		},
	}

	if err := fServer.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	peers := []io.Writer{}
	for _, peer := range fServer.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)

	if _, err := mw.Write([]byte{p2p.IncomingStreamByte}); err != nil {
		return err
	}
	if _, err := copyEncrypt(fServer.EncKey, fileBuf, mw); err != nil {
		return err
	}

	return nil
}

func (fServer *FileServer) Delete(key string) error {
	if err := fServer.store.Delete(key, fServer.ID); err != nil {
		return err
	}

	msg := &Message{
		ID: fServer.ID,
		Payload: MessageDeleteFile{
			Key: hashKey(key),
		},
	}

	return fServer.broadcast(msg)
}

type FileServerOpts struct {
	StoreOpts
	transport p2p.Transport

	EncKey         []byte
	BootStrapNodes []string
	ID             string
}

type FileServer struct {
	FileServerOpts

	store    *Store
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
		log.Printf("[%s] server stopped by user quit action\n", fServer.transport.Addr())
		if err := fServer.transport.Close(); err != nil {
			log.Printf("[%s] TCP transport closing error: %s\n", fServer.transport.Addr(), err)
		}
	}()

	for {
		select {
		case rpc := <-fServer.transport.Consume():
			var m *Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&m); err != nil {
				log.Printf("[%s] error decoding received message: %s\n", fServer.transport.Addr(), err)
			}

			if err := fServer.handleMessage(rpc.From, m); err != nil {
				log.Printf("[%s] error during handling message: %s\n", fServer.transport.Addr(), err)
				return
			}

		case <-fServer.quitch:
			return
		}
	}
}

func (fServer *FileServer) handleMessage(from string, m *Message) error {
	id := m.ID
	switch payload := m.Payload.(type) {
	case MessageStoreFile:
		if err := fServer.handleMessageStoreFile(from, id, &payload); err != nil {
			log.Printf("[%s] error storing streamed data: %s\n", fServer.transport.Addr(), err)
		}
	case MessageGetFile:
		if err := fServer.handleMessageGetFile(from, id, &payload); err != nil {
			log.Printf("[%s] error when streaming stored data: %s\n", fServer.transport.Addr(), err)
		}
	case MessageDeleteFile:
		if err := fServer.handleMessageDeleteFile(from, id, &payload); err != nil {
			log.Printf("[%s] error when streaming stored data: %s\n", fServer.transport.Addr(), err)
		}
	}

	return nil
}

func (fServer *FileServer) handleMessageGetFile(from string, id string, m *MessageGetFile) error {
	fileSize, r, err := fServer.store.Read(m.Key, id)
	if err != nil {
		return err
	}
	defer r.Close()

	peer, ok := fServer.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer %s not found in peer map", fServer.transport.Addr(), from)
	}

	if err := peer.Send([]byte{p2p.IncomingStreamByte}); err != nil {
		return err
	}
	binary.Write(peer, binary.LittleEndian, fileSize)
	_, err = io.Copy(peer, r)
	if err != nil {
		return err
	}

	return nil
}

func (fServer *FileServer) handleMessageStoreFile(from string, id string, m *MessageStoreFile) error {
	peer, ok := fServer.peers[from]
	defer peer.CloseStream()

	if !ok {
		return fmt.Errorf("[%s] peer %s not found in peer map", fServer.transport.Addr(), from)
	}

	if _, err := fServer.store.Write(m.Key, id, io.LimitReader(peer, m.Size)); err != nil {
		return err
	}

	return nil
}

func (fServer *FileServer) handleMessageDeleteFile(from string, id string, payload *MessageDeleteFile) error {
	if err := fServer.store.Delete(payload.Key, id); err != nil {
		return fmt.Errorf("[%s] could delete file of key (%s) for %s", fServer.transport.Addr(), payload.Key, from)
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
				log.Printf("[%s] error when dialing a remote node %s:, %s\n", fServer.transport.Addr(), address, err)
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
	log.Printf("[%s] remote peer added: %s\n", fServer.transport.Addr(), p.RemoteAddr().String())

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
}
