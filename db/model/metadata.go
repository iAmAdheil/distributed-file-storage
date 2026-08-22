package model

import (
	"bytes"
	"encoding/gob"
	"time"
)

type Metadata struct {
	ETag        string    `xml:"ETag"`
	Bucket      string    `xml:"-"`
	Key         string    `xml:"Key"`
	Size        int64     `xml:"Size"`
	ContentType string    `xml:"ContentType"`
	CreatedAt   time.Time `xml:"CreatedAt"`
}

func (md *Metadata) Encode(buf *bytes.Buffer) error {
	return gob.NewEncoder(buf).Encode(md)
}

func (md *Metadata) Decode(data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(&md)
}
