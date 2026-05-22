package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncryptDecryptFunc(t *testing.T) {
	payload := "This is a string"
	key := newEncryptionKey()
	buf := bytes.NewReader([]byte(payload))
	enc := new(bytes.Buffer)

	ne, err := copyEncrypt(key, buf, enc)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("encryption: ", ne)

	fmt.Println("Encrypted bytes: ", enc)

	out := new(bytes.Buffer)
	nd, err := copyDecrypt(key, enc, out)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("decryption: ", nd)

	if payload != out.String() {
		t.Error("decryption failed")
	}

	fmt.Println("Out: ", out.String())
}
