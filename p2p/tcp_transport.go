package p2p

import (
	"fmt"
	"net"
	"sync"
)

type TCPPeer struct {
	conn     net.Conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) TCPPeer {
	return TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (t TCPPeer) Close() error {
	return t.conn.Close()
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

func NewTCPTransport(opts TCPTransportOpts) TCPTransport {
	return TCPTransport{
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

// go routine wakes up to accept connections, call connection handling routine and then sleep
func (t *TCPTransport) startAndAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			continue
		}
		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		fmt.Println("dropping connection:", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, false)
	fmt.Println("new incoming connection:", peer)

	if err = t.Handshake(peer); err != nil {
		fmt.Println("TCP handshake failed:", err)
		return
	}

	if t.OnPeer(peer) != nil {
		if err = t.OnPeer(peer); err != nil {
			fmt.Println("TCP peer error:", err)
			return
		}
	}

	// start reading bytes in loop
	rpc := RPC{}
	for {
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			fmt.Println("TCP error:", err)
			return
		}

		rpc.From = conn.RemoteAddr()
		t.rpcch <- rpc
	}
}
