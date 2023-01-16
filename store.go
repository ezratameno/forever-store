package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

// CAS - content addressable.
// CASPathTransformFunc will turn the key into a specific path on disk.
func CASPathTransformFunc(key string) PathKey {

	// create sha from key.
	hash := sha1.Sum([]byte(key))

	// hash[:] convert from fix size into slice.
	hashStr := hex.EncodeToString(hash[:])

	// the length of every directory name on the path.
	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	// copy into the path the hashed string.
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from := i * blocksize
		to := (i * blocksize) + blocksize

		paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName:    strings.Join(paths, string(os.PathSeparator)),
		OriginalKey: hashStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName    string
	OriginalKey string
}

func (p *PathKey) Filename() string {
	return path.Join(p.PathName, p.OriginalKey)
}

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) string {
	return key
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathKey := s.PathTransformFunc(key)

	// create folder.
	if err := os.MkdirAll(pathKey.PathName, os.ModePerm); err != nil {
		return err
	}

	pathAndFileName := pathKey.Filename()

	// create the file.
	f, err := os.Create(pathAndFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// write to the file from the buffer.
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk", n)
	return nil

}
