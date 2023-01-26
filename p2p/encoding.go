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

	// Check the first byte to see the type message we are getting (stream/message).
	peekBuf := make([]byte, 1)
	_, err := r.Read(peekBuf)
	if err != nil {
		return err
	}

	// In case of a stream we are not decoding what is being sent over the network.
	// We are just setting stream true so we an handle that in our logic.
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}

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
