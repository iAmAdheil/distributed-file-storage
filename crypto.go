package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func newEncryptionKey() []byte {
	keyBuf := make([]byte, 16)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	var (
		stream = cipher.NewCTR(block, iv)
		buf    = make([]byte, 32*1024)
		nn     = 0
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n]) // encrypt and store in the same buf
			nw, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nn += nw
		}
		if err != io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return int64(nn), nil
}

func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := rand.Read(iv); err != nil {
		return 0, err
	}

	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	var (
		stream = cipher.NewCTR(block, iv)
		buf    = make([]byte, 32*1024)
		nn     = block.BlockSize()
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n]) // encrypt and store in the same buf
			nw, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nn += nw
		}
		if err != io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return int64(nn), nil
}
