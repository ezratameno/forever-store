package main

import (
	"bytes"
	"fmt"
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

func TestStore(t *testing.T) {

	s := newStore()
	defer teardown(t, s)
	count := 50
	id := generateID()
	// test multiple times with different keys.
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("foo_%d", i)

		// Write.
		data := []byte("some jpg bytes")
		_, err := s.writeStream(id, key, bytes.NewReader(data))
		assert.Nil(t, err)

		// Check that the file exists.
		assert.Equal(t, true, s.Has(id, key))

		// Read.
		_, r, err := s.Read(id, key)
		assert.Nil(t, err)
		b, err := io.ReadAll(r)
		assert.Nil(t, err)
		assert.Equal(t, data, b)

		assert.Nil(t, s.Delete(id, key))

		// Check that the file doesn't exists.
		assert.Equal(t, false, s.Has(id, key))
	}

}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	assert.Nil(t, s.Clear())
}
