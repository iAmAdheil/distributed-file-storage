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

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransportOpts struct {
	ListenAddress string
	Handshake     handshakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
	}
}

// init the listener
// start a separate thread for accepting connections
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
	peer := NewTCPPeer(conn, false)
	fmt.Println("new incoming connection:", peer)

	if err := t.Handshake(peer); err != nil {
		conn.Close()
		fmt.Println("TCP handshake failed:", err)
		return
	}

	// start reading bytes in loop
	msg := &Message{}
	for {
		if err := t.Decoder.Decode(conn, msg); err != nil {
			fmt.Println("TCP error:", err)
			return
		}

		msg.From = conn.RemoteAddr()

		fmt.Printf("received message: %+v\n", msg)
	}
}
