package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ezratameno/forever-store/p2p"
)

type FileServerOpts struct {
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {

	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key string

	// We need to specify the size of the message.
	// because we are streaming we won't get EOF, so we need to know how many bytes to read.
	Size int64
}

type MessageGetFile struct {
	Key string
}

// stream will send the Message to all the known peers in the network.
func (s *FileServer) stream(msg *Message) error {

	// peer implements the io.writer interface because the net.Conn implements it.
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	// transmit the data to all the peers.
	return gob.NewEncoder(mw).Encode(msg)
}

// broadcast will send the message to all the known peers in the network.
func (s *FileServer) broadcast(msg *Message) error {

	buf := new(bytes.Buffer)

	// Encode the msg.
	err := gob.NewEncoder(buf).Encode(msg)
	if err != nil {
		return err
	}

	// Send a message to all the peers telling what we want to do.
	for _, peer := range s.peers {

		// Send the first byte which will determine the type of incoming (Message/Stream)
		err := peer.Send([]byte{p2p.IncomingMessage})
		if err != nil {
			return err
		}

		// Send the message.
		err = peer.Send(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {

	// we have the key locally.
	if s.store.Has(key) {
		fmt.Printf("[%s] serving file (%s) locally from disk\n", s.Transport.Addr(), key)
		_, r, err := s.store.Read(key)
		return r, err
	}

	fmt.Printf("[%s] don't have file (%s) locally, fetching from network\n", s.Transport.Addr(), key)

	// ======================================================
	// Send a message to the known peers to get the key.
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	err := s.broadcast(&msg)
	if err != nil {
		return nil, err
	}

	time.Sleep(500 * time.Microsecond)

	// ==========================================
	// Read from every peer to get us the file.

	for _, peer := range s.peers {

		// First read the file size so we can limit the amount of bytes that we are reading from the connection,
		//  so it will not keep hanging.
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		// Store to disk the information we get from the peer.
		n, err := s.store.WriteDecrypt(s.EncKey, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)\n", s.Transport.Addr(), n, peer.RemoteAddr().String())
		peer.CloseStream()
	}

	_, r, err := s.store.Read(key)
	return r, err
}

// Store will store the data into disk and broadcast to all the known peers.
func (s *FileServer) Store(key string, r io.Reader) error {
	// 1. store this file to disk.
	// 2. broadcast this file to all known peers in the network.

	fileBuffer := new(bytes.Buffer)

	// once we read from the r the data will no longer be available.
	// tee helps us to make the data to be available also on the buffer.
	tee := io.TeeReader(r, fileBuffer)

	// ======================================
	// Store this file to our own disk.
	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	// ========================================
	// Broadcast the file to all the network to store.

	msg := Message{
		Payload: MessageStoreFile{
			Key: key,

			// Because of the iv we add to the file.
			Size: size + 16,
		},
	}

	err = s.broadcast(&msg)
	if err != nil {
		return err
	}

	// Stream the file we want to store.

	// Give the server some time to process the msg.
	time.Sleep(5 * time.Millisecond)

	// TODO: use a multiwriter here.

	peers := []io.Writer{}

	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	// send the type of operation we want to do.
	_, err = mw.Write([]byte{p2p.IncomingStream})
	if err != nil {
		return err
	}

	// stream the file.
	n, err := copyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	fmt.Printf("received and written (%d) bytes to disk \n", n)

	return nil
}

// OnPeer will add the peer to our network.
func (s *FileServer) OnPeer(p p2p.Peer) error {

	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	log.Printf("Connected with remote peer: %s", p.RemoteAddr().String())
	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

// loop will read messages.
func (s *FileServer) loop() {

	defer func() {
		log.Println("file server stopped due error or user quit action.")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():

			var msg Message

			// decode into a Message.
			err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg)
			if err != nil {
				log.Printf("decoding error: %+v\n", err)
				continue
			}

			err = s.handleMessage(rpc.From, &msg)
			if err != nil {
				log.Printf("handle message error: %+v\n", err)
				continue
			}

		case <-s.quitch:
			return
		}
	}
}

// handleMessage will handle the message according to the underlying type.
func (s *FileServer) handleMessage(from string, msg *Message) error {

	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)

	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

// handleMessageGetFile will send over the wire the desired file.
func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {

	if !s.store.Has(msg.Key) {
		return fmt.Errorf("[%s] need to serve file (%s), but it does not exist on disk", s.Transport.Addr(), msg.Key)
	}

	fmt.Printf("serving file (%s) over the network \n", msg.Key)

	// Fetch the file.
	fileSize, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	// Check if it's a reader closer then close.
	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
	}

	// Send the reader over the wire to the one who sent the message.
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer map", from)
	}

	// First let the peer know it's a streaming operation.
	err = peer.Send([]byte{p2p.IncomingStream})
	if err != nil {
		return err
	}

	// Send the file size as an int64
	binary.Write(peer, binary.LittleEndian, fileSize)

	// Stream the file over the wire.
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] served file (%s) of size (%d) bytes over the network to %s\n", s.Transport.Addr(), msg.Key, n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer map", from)
	}

	// Tell how many bytes we want to read from the peer.
	r := io.LimitReader(peer, msg.Size)

	// Store the file into the disk from the peer who sent the message.
	n, err := s.store.Write(msg.Key, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written (%d) bytes to disk\n", s.Transport.Addr(), n)

	// Enable to continue reading from the peer.
	peer.CloseStream()

	return nil
}

// BootstrapNetwork will add default nodes to our network.
func (s *FileServer) BootstrapNetwork() error {

	for _, addr := range s.BootstrapNodes {

		fmt.Printf("Attempting to connect with remote: %s \n", addr)

		go func(addr string) {

			err := s.Transport.Dial(addr)
			if err != nil {
				log.Printf("dial error: %+v", err)
				return
			}

			fmt.Printf("Connected with remote: %s \n", addr)

		}(addr)
	}

	return nil
}

func (s *FileServer) Start() error {

	err := s.Transport.ListenAndAccept()
	if err != nil {
		return err
	}

	s.BootstrapNetwork()
	s.loop()
	return nil

}

func init() {

	// we need to register all the types we want to send in the message.payload as any.
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})

}
