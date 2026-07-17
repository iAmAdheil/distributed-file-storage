package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/db/model"
)

func TestDB(t *testing.T) {
	dbOpts := &DBOpts{
		Filename: "test_db.db",
	}

	testMd := &model.Metadata{
		Bucket:      "testbucket",
		Key:         "testkey",
		Size:        100,
		ContentType: "text",
		CreatedAt:   time.Now(),
	}

	db := NewDB(*dbOpts)
	if err := db.PutMeta(testMd); err != nil {
		t.Errorf("Adding the object failed: %s\n", err.Error())
	}

	md, err := db.GetMeta(testMd.Bucket, testMd.Key)
	if err != nil {
		t.Errorf("Adding the object failed: %s\n", err.Error())
	}

	fmt.Println("Received Metadata:", *md)

	if err := db.DeleteMeta(testMd.Bucket, testMd.Key); err != nil {
		t.Errorf("Deleting the object failed: %s\n", err.Error())
	}
}
