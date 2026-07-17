package model

import (
	"bytes"
	"encoding/gob"
	"time"
)

type Metadata struct {
	Bucket      string
	Key         string
	Size        int64
	ContentType string
	CreatedAt   time.Time
}

func (md *Metadata) Encode(buf *bytes.Buffer) error {
	return gob.NewEncoder(buf).Encode(md)
}

func (md *Metadata) Decode(data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(&md)
}
