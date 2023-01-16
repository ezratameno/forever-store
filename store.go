package main

import (
	"bytes"
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
		PathName: strings.Join(paths, string(os.PathSeparator)),
		Filename: hashStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	Filename string
}

// FirstPathName returns the root path name.
func (p *PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, string(os.PathSeparator))
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}
func (p *PathKey) FullPath() string {
	return path.Join(p.PathName, p.Filename)
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

// Has checks if we have the file exists.
func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	pathAndFileName := pathKey.FullPath()

	_, err := os.Stat(pathAndFileName)

	return err == nil

}

// Delete deletes a file from the disk.
func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	pathAndFileName := pathKey.FullPath()

	defer func() {
		log.Printf("deleted [%s] from disk", pathAndFileName)
	}()

	// delete the root directory recursively.
	return os.RemoveAll(pathKey.FirstPathName())
}

// Read reads the content of the file.
func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	pathAndFileName := pathKey.FullPath()

	return os.Open(pathAndFileName)
}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathKey := s.PathTransformFunc(key)

	// create folder.
	if err := os.MkdirAll(pathKey.PathName, os.ModePerm); err != nil {
		return err
	}

	pathAndFileName := pathKey.FullPath()

	// create the file.
	f, err := os.Create(pathAndFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// write to the file from the reader.
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk", n)
	return nil

}
