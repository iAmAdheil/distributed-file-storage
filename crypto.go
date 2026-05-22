package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
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

func hashKey(key string) string {
	buf := md5.Sum([]byte(key))
	return hex.EncodeToString(buf[:])
}

func newEncryptionKey() []byte {
	keyBuf := make([]byte, 16)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyStream(stream cipher.Stream, blocksize int, src io.Reader, dst io.Writer) (int64, error) {
	var (
		buf = make([]byte, 32*1024)
		nn  = blocksize
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

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
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

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}
