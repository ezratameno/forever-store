package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathTransformFunc(t *testing.T) {
	key := "mombestpicture"
	pathKey := CASPathTransformFunc(key)

	assert.Equal(t, pathKey.PathName, "cf5d4/b01c4/d9438/c22c5/6c832/f83bd/3e8c6/304f9")
	assert.Equal(t, pathKey.Filename, "cf5d4b01c4d9438c22c56c832f83bd3e8c6304f9")
}

func TestDelete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(opts)
	key := "momsspecials"

	// write.
	data := []byte("some jpg bytes")
	assert.Nil(t, s.writeStream(key, bytes.NewReader(data)))

	// check that the file exist.
	assert.Equal(t, true, s.Has(key))

	assert.Nil(t, s.Delete(key))

	// check that the file doesn't exist.
	assert.Equal(t, false, s.Has(key))
}
func TestStore(t *testing.T) {

	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(opts)
	key := "momsspecials"

	// write.
	data := []byte("some jpg bytes")
	assert.Nil(t, s.writeStream(key, bytes.NewReader(data)))

	// read.
	r, err := s.Read(key)

	assert.Nil(t, err)

	b, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, data, b)

	s.Delete(key)

}
