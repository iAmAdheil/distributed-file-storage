package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

type TCPPeer struct {
	net.Conn
	outbound bool
	Wg       *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

type TCPTransportOpts struct {
	ListenAddress string
	Handshake     handshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

// implements the Transport interface
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// init the listener
// start a separate thread for accepting connections
// implements the Transport interface
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return err
	}
	fmt.Println("listening on:", t.ListenAddress+", yaaay!")
	go t.startAndAcceptLoop()
	return nil
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	log.Println("dialing peer: ", addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn)
	return nil
}

// go routine wakes up to accept connections, call connection handling routine and then sleep
func (t *TCPTransport) startAndAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Println("TCP accept error: ", err)
		}
		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		log.Println("dropping connection: ", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, false)

	if err = t.Handshake(peer); err != nil {
		log.Println("TCP handshake failed: ", err)
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			log.Println("TCP peer error: ", err)
			return
		}
	}

	// start reading bytes in loop
	rpc := RPC{}
	for {
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			log.Println("TCP error: ", err)
			return
		}

		rpc.From = conn.RemoteAddr().String()
		peer.Wg.Add(1)
		t.rpcch <- rpc
		fmt.Println("waiting for stream to end")
		peer.Wg.Wait()
		fmt.Println("normal functioning started")
	}
}
