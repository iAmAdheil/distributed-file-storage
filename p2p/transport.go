package p2p

import "io"

type Peer interface{}

type Transport interface {
	ListenAndAccept() error
	Handshake(Peer) error
	Decode(r io.Reader, m *Message) error
}
