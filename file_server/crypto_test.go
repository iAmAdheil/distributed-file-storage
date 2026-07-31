package file_server

import (
	"bytes"
	"fmt"
	"testing"
)

func TestHashKey(t *testing.T) {
	key := "Key2"
	res := hashKey(key)
	expectedRes := "c31650e34686537020ee36190e71da5b"
	if res != expectedRes {
		t.Error("key hashing failed")
	}
}

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
