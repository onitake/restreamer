package restreamer

import (
	"log"
	"time"
	"errors"
	"net/http"
	"encoding/hex"
)

const (
	DEFAULT_TIMEOUT time.Duration = 10 * time.Second
)

var (
	// ErrNoConnection is thrown when trying to read
	// from a stream that is not connected
	ErrNoConnection = errors.New("restreamer: socket not connected")
	// ErrNoConnection is thrown when trying to
	// connect to an already established upstream socket
	ErrAlreadyConnected = errors.New("restreamer: socket is already connected")
	// ErrInvalidResponse is thrown when an unsupported
	// HTTP response code was received
	ErrInvalidResponse = errors.New("restreamer: unsupported response code")
	// ErrQueueFull is thrown when more data is available
	// than the input queue can handle
	ErrQueueFull = errors.New("restreamer: queue full")
	// ErrQueueFull is thrown when trying to process
	// data while none is available
	ErrQueueEmpty = errors.New("restreamer: queue empty")
)

// a streaming HTTP client
type Client struct {
	// the URL to GET
	Url string
	// the response, including the body reader
	Socket *http.Response
	// the I/O timeout
	Timeout time.Duration
	// the packet queue
	Queue chan<- Packet
	// true while the client is streaming into the queue
	Running bool
}

// construct a new streaming HTTP client
// the client is still unconnected, you
// need to call Connect next
func NewClient(url string, queue chan<- Packet) *Client {
	return &Client {
		Url: url,
		Socket: nil,
		Timeout: DEFAULT_TIMEOUT,
		Queue: queue,
		Running: false,
	}
}

// connects the socket, sends the HTTP request
// and spawns the streaming thread
func (client *Client) Connect() error {
	if !client.Running {
		getter := &http.Client {
			Timeout: client.Timeout,
		}
		response, err := getter.Get(client.Url)
		if err != nil {
			return err
		}
		client.Socket = response
		client.Running = true
		go client.pull()
		return nil
	}
	return ErrAlreadyConnected
}

// closes the connection
func (client *Client) Close() error {
	if client.Socket != nil {
		err := client.Socket.Body.Close()
		client.Socket = nil
		return err
	}
	return ErrNoConnection
}

// returns the HTTP status code or 0 if not connected
func (client *Client) StatusCode() int {
	if client.Socket != nil {
		return client.Socket.StatusCode
	}
	return 0
}

// returns the HTTP status message or the empty string if not connected
func (client *Client) Status() string {
	if client.Socket != nil {
		return client.Socket.Status
	}
	return ""
}

// returns true if the socket is connected
func (client *Client) Connected() bool {
	return client.Running
}

// streams data from the socket to the queue
func (client *Client) pull() {
	log.Printf("Reading stream from %s\n", client.Url)
	
	for client.Running {
		packets, err := ReadPacket(client.Socket.Body)
		if err != nil {
			log.Printf("Got error on stream %s: %s\n", client.Url, err)
			client.Running = false
		} else {
			for _, packet := range packets {
				log.Printf("Got a packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				client.Queue<- packet
			}
		}
	}
	
	log.Printf("Socket for stream %s closed, exiting\n", client.Url)
	client.Socket.Body.Close()
	client.Socket = nil
}

// connects the data source
/*
func (s *Stream) Connect() error {
	log.Printf("Connecting to %s\n", s.Source.Url)

	err := s.Source.Connect()
	if err != nil {
		return err
	}
	if s.Source.StatusCode() != http.StatusOK && s.Source.StatusCode() != http.StatusPartialContent {
		return ClientError { fmt.Sprintf("Unsupported response code: %s", s.Source.Status()) }
	}
	
	// start streaming
	go s.Pull()
	
	return nil
}
*/

