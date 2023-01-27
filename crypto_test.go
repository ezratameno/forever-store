package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "Foo not Bar"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := newEncryptionKey()

	_, err := copyEncrypt(key, src, dst)
	assert.Nil(t, err)
	fmt.Printf("encrypted: %s\n", dst.String())

	out := new(bytes.Buffer)

	nw, err := copyDecrypt(key, dst, out)
	assert.Nil(t, err)

	// +16 because of the iv.
	assert.Equal(t, len(payload)+16, nw)
	assert.Equal(t, payload, out.String())

	fmt.Printf("decrypted: %s\n", out.String())

}
