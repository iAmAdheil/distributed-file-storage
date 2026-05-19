package p2p

import "net"

type Peer interface {
	net.Conn
	Send(b []byte) error
}

type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(string) error
}
