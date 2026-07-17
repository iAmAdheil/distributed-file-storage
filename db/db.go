package db

import (
	"bytes"
	"fmt"

	"github.com/iAmAdheil/distributed-file-storage/db/model"
	bolt "go.etcd.io/bbolt"
)

type DBOpts struct {
	Filename string
}

type DB struct {
	in *bolt.DB
}

func NewDB(dbOpts DBOpts) *DB {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(dbOpts.Filename, 0600, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("db:(%s) has been opened.\n", dbOpts.Filename)

	return &DB{
		in: db,
	}
}

func (db *DB) PutMeta(md *model.Metadata) error {
	err := db.in.Update(func(tx *bolt.Tx) error {
		bname := []byte(md.Bucket)
		k := []byte(md.Key)
		bucket, err := tx.CreateBucketIfNotExists(bname)
		if err != nil {
			return err
		}

		val := bytes.NewBuffer([]byte{})
		if err := md.Encode(val); err != nil {
			return err
		}

		bucket.Put(k, val.Bytes())

		fmt.Printf("Object added to bucket (%s)\n", md.Bucket)

		return nil
	})

	return err
}

func (db *DB) DeleteMeta(Bucket string, Key string) error {
	err := db.in.Update(func(tx *bolt.Tx) error {
		bname := []byte(Bucket)
		k := []byte(Key)

		bucket := tx.Bucket(bname)
		if bucket == nil {
			return fmt.Errorf("Bucket (%s) does not exist.", Bucket)
		}

		if err := bucket.Delete(k); err != nil {
			return err
		}

		fmt.Printf("Object deleted from bucket (%s)\n", Bucket)

		return nil
	})

	return err
}

func (db *DB) GetMeta(Bucket string, Key string) (*model.Metadata, error) {
	md := &model.Metadata{}

	err := db.in.View(func(tx *bolt.Tx) error {
		bname := []byte(Bucket)
		k := []byte(Key)

		bucket := tx.Bucket(bname)
		if bucket == nil {
			return fmt.Errorf("Bucket (%s) does not exist.", Bucket)
		}

		data := bucket.Get(k)
		if len(data) == 0 {
			return fmt.Errorf("Key (%s) not found.", Key)
		}

		if err := md.Decode(data); err != nil {
			return err
		}

		return nil
	})

	return md, err
}
