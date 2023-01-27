package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

const (
	defaultRootFolderName = "ggnetwork"
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

	// copy into the paths the hashed string in length of blocksize.
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

// FullPath returns the full path to the file.
func (p *PathKey) FullPath() string {
	return path.Join(p.PathName, p.Filename)
}

type StoreOpts struct {

	// Root is the folder name of the root, containing all the folders/files of the system.
	Root string

	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {

	// Defaults.
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if opts.Root == "" {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

// Has checks if we have the file exists.
func (s *Store) Has(id, key string) bool {
	pathKey := s.PathTransformFunc(key)
	pathAndFileName := path.Join(s.Root, id, pathKey.FullPath())

	_, err := os.Stat(pathAndFileName)
	return !errors.Is(err, os.ErrNotExist)
}

// Clear removes all the files from the root.
func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// Delete deletes a file from the disk.
func (s *Store) Delete(id, key string) error {
	pathKey := s.PathTransformFunc(key)
	pathAndFileName := pathKey.FullPath()

	defer func() {
		log.Printf("deleted [%s] from disk", pathAndFileName)
	}()

	// delete the root directory recursively.
	return os.RemoveAll(path.Join(s.Root, id, pathKey.FirstPathName()))
}

// Write writes the file to disk and returns the number of bytes written.
func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

// writeStream writes a file into the disk.
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {

	// create the file.
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	defer f.Close()

	// write to the file from the reader.
	return io.Copy(f, r)

}

func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {

	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	defer f.Close()

	// write to the file from the reader.
	n, err := copyDecrypt(encKey, r, f)

	return int64(n), err
}

// Read reads the content of the file.
func (s *Store) Read(id, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

func (s *Store) readStream(id, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	pathAndFileName := path.Join(s.Root, id, pathKey.FullPath())

	// get the file size.
	fi, err := os.Stat(pathAndFileName)
	if err != nil {
		return 0, nil, err
	}

	file, err := os.Open(pathAndFileName)
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (s *Store) openFileForWriting(id, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := path.Join(s.Root, id, pathKey.PathName)
	// create folder under the root.
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	pathAndFileName := path.Join(s.Root, id, pathKey.FullPath())

	// create the file.
	return os.Create(pathAndFileName)
}
