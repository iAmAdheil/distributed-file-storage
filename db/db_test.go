package db

import (
	"fmt"
	"net/http"
	"strconv"
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
		CreatedAt:   time.Now().UTC().Format(http.TimeFormat),
	}

	db := New(*dbOpts)
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

func TestListMeta(t *testing.T) {
	dbOpts := &DBOpts{
		Filename: "test_db.db",
	}

	db := New(*dbOpts)

	for i := 0; i < 10; i++ {
		testMd := &model.Metadata{
			Bucket:      "testbucket",
			Key:         "testkey_" + strconv.Itoa(i),
			Size:        100,
			ContentType: "text",
			CreatedAt:   time.Now().UTC().Format(http.TimeFormat),
		}
		if err := db.PutMeta(testMd); err != nil {
			t.Errorf("Adding the object failed: %s\n", err.Error())
		}
	}

	res, err := db.ListMeta("testbucket", ListMetaParams{
		ContToken: "testkey_2",
		MaxKeys:   2,
	})
	if err != nil {
		t.Errorf("Listing bucket items failed: %s\n", err.Error())
	}

	fmt.Printf("List items: %v\n", res.List)
	fmt.Println("Next continuation token:", res.NextContToken)

	for i := 0; i < 10; i++ {
		if err := db.DeleteMeta("testbucket", "testkey_"+strconv.Itoa(i)); err != nil {
			t.Errorf("Deleting the object failed: %s\n", err.Error())
		}
	}
}
