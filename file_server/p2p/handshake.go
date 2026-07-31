package p2p

type handshakeFunc func(Peer) error

// handle handshake between our server and the peer
func NOPHandshakeFunc(Peer) error { return nil }
