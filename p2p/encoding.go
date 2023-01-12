package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {

	buf := make([]byte, 1028)

	// Read the bytes into the buffer.
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	// Place all read bytes into the message.
	msg.Payload = buf[:n]
	return nil
}
