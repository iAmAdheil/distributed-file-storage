package main

import (
	"log"

	"github.com/iAmAdheil/distributed-file-storage/p2p"
)

func main() {
	opts := p2p.TCPTransportOpts{
		ListenAddress: ":3000",
		Handshake:     p2p.NOPHandshakeFunc,
		Decoder:       &p2p.DefaultDecoder{},
	}
	tr := p2p.NewTCPTransport(opts)

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal("some tcp error ahhhh:", err)
	}

	select {}
}
