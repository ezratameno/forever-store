package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RPC holds any arbitrary data that is being sent
//
//	over each transport between two nodes in the network.
type RPC struct {
	Payload []byte
	From    string

	// Stream will let us know if we are streaming.
	// if we are streaming we need to lock and unlock when the stream is done,
	//  so we won't receive other messages in the read loop.
	Stream bool
}
