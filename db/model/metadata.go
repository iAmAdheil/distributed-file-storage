package model

import (
	"bytes"
	"encoding/gob"
)

type Metadata struct {
	Bucket      string
	Key         string
	Size        int
	ContentType string
	CreatedAt   string
}

func (md *Metadata) Encode(buf *bytes.Buffer) error {
	return gob.NewEncoder(buf).Encode(md)
}

func (md *Metadata) Decode(buf *bytes.Buffer) error {
	return gob.NewDecoder(buf).Decode(&md)
}
