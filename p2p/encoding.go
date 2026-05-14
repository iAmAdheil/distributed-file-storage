package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(r io.Reader, m *Message) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, m *Message) error {
	return gob.NewDecoder(r).Decode(m)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, m *Message) error {
	buf := make([]byte, 2000)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	m.Payload = buf[:n]
	return nil
}
