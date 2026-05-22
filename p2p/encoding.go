package p2p

import (
	"encoding/gob"
	"fmt"
	"io"
)

type Decoder interface {
	Decode(r io.Reader, m *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, m *RPC) error {
	return gob.NewDecoder(r).Decode(m)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, m *RPC) error {
	peekBuf := make([]byte, 1)
	_, err := r.Read(peekBuf)
	if err != nil {
		fmt.Println("ddd")
		return err
	}
	if peekBuf[0] == IncomingStreamByte {
		m.Stream = true
		return nil
	}

	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	m.Payload = buf[:n]
	m.Stream = false
	return nil
}
