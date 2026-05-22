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

func (p *TCPPeer) CloseStream() {
	p.Wg.Done()
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

func (t *TCPTransport) Addr() string {
	return t.ListenAddress
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
	fmt.Println("Listening on:", t.ListenAddress, ", yaaay!")
	go t.startAndAcceptLoop()
	return nil
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	log.Printf("[%s] dialing peer: %s\n", t.ListenAddress, addr)

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
			log.Printf("[%s] TCP accept error: %s\n", t.ListenAddress, err)
		}
		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	peer := NewTCPPeer(conn, false)

	defer func() {
		log.Printf("[%s] dropping connection with [%s]: %s\n", t.ListenAddress, peer.RemoteAddr(), err)
		conn.Close()
	}()

	if err = t.Handshake(peer); err != nil {
		log.Printf("[%s] TCP handshake failed with [%s]: %s\n", t.ListenAddress, peer.RemoteAddr(), err)
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			log.Printf("[%s] TCP peer error: %s\n", t.ListenAddress, err)
			return
		}
	}

	// start reading bytes in loop
	for {
		rpc := RPC{}
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			log.Printf("[%s] TCP decoder error, sent from [%s]: %s\n", t.ListenAddress, conn.RemoteAddr(), err)
			return
		}

		rpc.From = conn.RemoteAddr().String()
		if rpc.Stream {
			peer.Wg.Add(1)
			fmt.Printf("[%s] blocked for direct streaming to server from [%s]\n", t.ListenAddress, rpc.From)
			peer.Wg.Wait()
			fmt.Printf("[%s] streaming complete, read loop resumes\n", t.ListenAddress)
			continue
		}

		t.rpcch <- rpc
	}
}
