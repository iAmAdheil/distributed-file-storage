package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

func genID() string {
	buf := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func newEncryptionKey() []byte {
	keyBuf := make([]byte, 16)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}
