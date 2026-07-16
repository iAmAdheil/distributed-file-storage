package db

import (
	"testing"

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
		CreatedAt:   "test-time",
	}

	db := NewDB(*dbOpts)
	if err := db.PutMeta(testMd); err != nil {
		t.Errorf("Adding the object failed: %s\n", err.Error())
	}

	if err := db.DeleteMeta(testMd.Bucket, testMd.Key); err != nil {
		t.Errorf("Deleting the object failed: %s\n", err.Error())
	}
}
