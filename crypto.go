package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

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

	// Maximum amount we will copy into memory.
	// in streaming we don't need to copy to disk every byte.
	buf := make([]byte, 32*1024)

	stream := cipher.NewCTR(block, iv)
	nw := block.BlockSize()

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}

			// Add the number of bytes written.
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

	// Maximum amount we will copy into memory.
	// in streaming we don't need to copy to disk every byte.
	buf := make([]byte, 32*1024)

	stream := cipher.NewCTR(block, iv)
	nw := block.BlockSize()
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
