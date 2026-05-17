package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRoot = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i+1)*blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

func DefaultPathTransformFunc(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathname string
	Filename string
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.Filename)
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRoot
	}
	return &Store{StoreOpts: opts}
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Has(key string) bool {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	fullPath := pathKey.FullPath()
	fullPathWithRoot := s.Root + "/" + fullPath
	_, err := os.Stat(fullPathWithRoot)
	return !os.IsNotExist(err)
}

func (s *Store) Delete(key string) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk\n", pathKey.Filename)
	}()

	deletePath := s.Root + "/" + pathKey.FirstPathName()

	if err := os.RemoveAll(deletePath); err != nil {
		return err
	}
	return nil
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, f); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	fullPath := pathKey.FullPath()
	fullPathWithRoot := s.Root + "/" + fullPath
	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	pathnameWithRoot := s.Root + "/" + pathKey.Pathname
	if err := os.MkdirAll(pathnameWithRoot, os.ModePerm); err != nil {
		return err
	}

	fullPathWithRoot := s.Root + "/" + pathKey.FullPath()
	file, err := os.Create(fullPathWithRoot)
	if err != nil {
		return err
	}

	n, err := io.Copy(file, r)
	if err != nil {
		return err
	}

	log.Printf("wrote %d bytes to %s\n", n, fullPathWithRoot)

	return nil
}
