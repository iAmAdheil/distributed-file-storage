package db

import (
	"bytes"
	"fmt"
	"strings"

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
			return fmt.Errorf("Bucket (%s) does not exist.\n", Bucket)
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
			return &BucketNotFound{Name: Bucket}
		}

		data := bucket.Get(k)
		if len(data) == 0 {
			return &KeyNotFound{Name: Key}
		}

		if err := md.Decode(data); err != nil {
			return err
		}

		return nil
	})

	return md, err
}

type ListMetaParams struct {
	Prefix    string // filter out entries starting with prefix
	MaxKeys   int
	ContToken string // start from after this token
}

type ListMetaRes struct {
	List      []model.Metadata
	ContToken string
}

func (db *DB) ListMeta(Bucket string, params ListMetaParams) (*ListMetaRes, error) {
	list := []model.Metadata{}
	contToken := ""

	err := db.in.View(func(tx *bolt.Tx) error {
		bname := []byte(Bucket)

		bucket := tx.Bucket(bname)
		if bucket == nil {
			return &BucketNotFound{Name: Bucket}
		}

		count := params.MaxKeys
		if count <= 0 {
			count = 1000 // sane limit
		}

		cursor := bucket.Cursor()
		// set up the starting pos, while also adding it
		var k []byte
		var v []byte
		t := []byte(params.ContToken)
		if len(params.ContToken) > 0 {
			k, v = cursor.Seek(t)
		} else {
			k, v = cursor.First()
		}
		if len(v) > 0 && strings.HasPrefix(string(k), params.Prefix) {
			var md model.Metadata
			if err := (&md).Decode(v); err != nil {
				return err
			}
			list = append(list, md)
			count--
		}

		for {
			k, v := cursor.Next()
			if k == nil || v == nil {
				break
			}

			// Does 2 things ->
			// 1. prevents non-prefix matching keys to be added to list
			// 2. check if bucket still has entries left with matching prefix
			// after last added element to the list
			if !strings.HasPrefix(string(k), params.Prefix) {
				continue
			}

			// track if elements after last entry still have same prefix
			if count == 0 {
				if len(list) > 0 {
					last := list[len(list)-1]
					contToken = last.Key
				}
				break
			}

			// never add elements once max keys = 0
			if count > 0 {
				var md model.Metadata
				if err := (&md).Decode(v); err != nil {
					return err
				}
				list = append(list, md)
				count--
			}
		}

		return nil
	})

	res := &ListMetaRes{
		List:      list,
		ContToken: contToken,
	}

	return res, err
}
