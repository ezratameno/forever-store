package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// generateID will generate a random id.
func generateID() string {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

func hashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func newEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Read the IV from the given io.Reader which, in our case should be the block.BlockSize() bytes we read.

	iv := make([]byte, block.BlockSize()) // 16 bytes.

	_, err = src.Read(iv)
	if err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}

func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {

	// Maximum amount we will copy into memory.
	// in streaming we don't need to copy to disk every byte.
	buf := make([]byte, 32*1024)

	nw := blockSize

	for {
		// read everything from the src into the buffer.
		n, err := src.Read(buf)
		if n > 0 {
			// write every read byte into the buffer as XOR.
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}

			// Add the number of bytes read.
			nw += nn
		}

		// Done reading.
		if err == io.EOF {
			break
		}

		if err != nil {
			return 0, err
		}
	}

	return nw, nil
}

// copyEncrypt will copy and encrypt and return the number of bytes written.
func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) // 16 bytes.

	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return 0, err
	}

	// Prepend the IV to the file.
	_, err = dst.Write(iv)
	if err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}
