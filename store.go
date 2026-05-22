package main

import (
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

func (s *Store) Has(key string, id string) bool {
	fullPathWithRoot := s.getPath(key, id)
	_, err := os.Stat(fullPathWithRoot)
	return !os.IsNotExist(err)
}

func (s *Store) Delete(key string, id string) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)

	defer func() {
		log.Printf("delete [%s] from disk\n", pathKey.Filename)
	}()

	deletePath := s.Root + "/" + id + "/" + pathKey.FirstPathName()

	if err := os.RemoveAll(deletePath); err != nil {
		return err
	}
	return nil
}

func (s *Store) Write(key string, id string, r io.Reader) (int64, error) {
	return s.writeStream(key, id, r)
}

func (s *Store) DWrite(encKey []byte, key string, id string, r io.Reader) (int64, error) {
	return s.decryptWriteStream(encKey, key, id, r)
}

func (s *Store) decryptWriteStream(encKey []byte, key string, id string, r io.Reader) (int64, error) {
	file, fullPathWithRoot, err := s.openFileForWrite(key, id)
	n, err := copyDecrypt(encKey, r, file)
	if err != nil {
		return 0, err
	}
	log.Printf("performed decryption and wrote [%d] bytes to %s\n", n, fullPathWithRoot)

	return n, nil
}

func (s *Store) writeStream(key string, id string, r io.Reader) (int64, error) {
	file, fullPathWithRoot, err := s.openFileForWrite(key, id)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(file, r)
	if err != nil {
		return 0, err
	}

	log.Printf("wrote [%d] bytes to %s\n", n, fullPathWithRoot)

	return n, nil
}

func (s *Store) openFileForWrite(key string, id string) (*os.File, string, error) {
	err := s.makePath(key, id)
	if err != nil {
		return nil, "", err
	}
	fullPathWithRoot := s.getPath(key, id)
	file, err := os.Create(fullPathWithRoot)
	if err != nil {
		return nil, "", err
	}

	return file, fullPathWithRoot, nil
}

func (s *Store) Read(key string, id string) (int64, io.ReadCloser, error) {
	return s.readStream(key, id)
}

func (s *Store) readStream(key string, id string) (int64, io.ReadCloser, error) {
	fullPathWithRoot := s.getPath(key, id)
	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}
	fs, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}
	return fs.Size(), file, nil
}

func (s *Store) getPath(key string, id string) string {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	fullPathWithRoot := s.Root + "/" + id + "/" + pathKey.FullPath()
	return fullPathWithRoot
}

func (s *Store) makePath(key string, id string) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	pathnameWithRoot := s.Root + "/" + id + "/" + pathKey.Pathname
	return os.MkdirAll(pathnameWithRoot, os.ModePerm)
}
