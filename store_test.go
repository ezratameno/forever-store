package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathTransformFunc(t *testing.T) {
	key := "mombestpicture"
	pathKey := CASPathTransformFunc(key)

	assert.Equal(t, pathKey.PathName, "cf5d4/b01c4/d9438/c22c5/6c832/f83bd/3e8c6/304f9")
	assert.Equal(t, pathKey.OriginalKey, "cf5d4b01c4d9438c22c56c832f83bd3e8c6304f9")
}

func TestStore(t *testing.T) {

	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("some jpg bytes"))

	assert.Nil(t, s.writeStream("myspecialpicture", data))
}
