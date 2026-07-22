package db

import "fmt"

type BucketNotFound struct {
	Name string
}

type KeyNotFound struct {
	Name string
}

func (b *BucketNotFound) Error() string {
	return fmt.Sprintf("Bucket (%s) does not exist.\n", b.Name)
}

func (k *KeyNotFound) Error() string {
	return fmt.Sprintf("Key (%s) not found.\n", k.Name)
}
