package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "Pagani Huayra"
	pathname := CASPathTransformFunc(key)
	expectedOriginal := "47a7b0693f9fca1237594eb7ae798aa590940eef"
	expectedPathname := "47a7b/0693f/9fca1/23759/4eb7a/e798a/a5909/40eef"

	if pathname.Pathname != expectedPathname {
		t.Errorf("expected pathname %s, got %s", expectedPathname, pathname.Pathname)
	}
	if pathname.Filename != expectedOriginal {
		t.Errorf("expected original %s, got %s", expectedOriginal, pathname.Filename)
	}
}

func TestStore(t *testing.T) {
	store := newStore()
	key_pref := "Pagani_Huayra"

	defer func() {
		if err := purge(store); err != nil {
			t.Errorf("purge failed")
		}
	}()

	for i := 0; i < 50; i++ {
		key := key_pref + "_" + strconv.Itoa(i)
		data := []byte("The pagani huayra is my favourite car")
		dataReader := bytes.NewReader(data)
		size := int64(len(data))

		if _, err := store.Write(key, "1", io.LimitReader(dataReader, size)); err != nil {
			t.Errorf("expected no error, got %v\n", err)
			return
		}

		if ok := store.Has(key, "1"); !ok {
			t.Errorf("expected to have key %s\n", key)
			return
		}

		_, r, err := store.Read(key, "1")
		if err != nil {
			t.Errorf("expected no error, got %v\n", err)
			return
		}
		b, err := io.ReadAll(r)
		fmt.Printf("data in file with key %s: %s\n", key, string(b))

		if err := store.Delete(key, "1"); err != nil {
			t.Errorf("path could not be deleted for key: %s\n", key)
			return
		}

		if ok := store.Has(key, "1"); ok {
			t.Errorf("expected to NOT have key %s\n", key)
			return
		}
	}
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	store := NewStore(opts)
	return store
}

func purge(s *Store) error {
	return s.Clear()
}
