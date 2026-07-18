package model

import (
	"bytes"
	"encoding/gob"
	"time"
)

type Metadata struct {
	Bucket      string    `json:"bucket"`
	Key         string    `json:"key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (md *Metadata) Encode(buf *bytes.Buffer) error {
	return gob.NewEncoder(buf).Encode(md)
}

func (md *Metadata) Decode(data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(&md)
}
