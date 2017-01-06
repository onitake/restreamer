package restreamer

import (
	"os"
	"io"
	"log"
	"time"
	"errors"
	"net/http"
	"net/url"
)

const (
	DefaultTimeout time.Duration = 10 * time.Second
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
	Url *url.URL
	// the response, including the body reader
	socket *http.Response
	// the input stream (socket)
	input io.ReadCloser
	// the I/O timeout
	Timeout time.Duration
	// the packet queue
	queue chan<- Packet
	// true while the client is streaming into the queue
	running bool
}

// construct a new streaming HTTP client,
// without connecting the socket yet.
// you need to call Connect() to do that.
// after a connection has been closed,
// the client can not be reused and must
// be cloned and restarted.
func NewClient(uri string, queue chan<- Packet, timeout int) (*Client, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return &Client {
		Url: parsed,
		socket: nil,
		input: nil,
		Timeout: time.Duration(timeout) * time.Second,
		queue: queue,
		running: false,
	}, nil
}

// connects the socket, sends the HTTP request
// and spawns the streaming thread
func (client *Client) Connect() error {
	if client.input == nil {
		if client.Url.Scheme == "file" {
			log.Printf("Opening %s\n", client.Url.Path)
			file, err := os.Open(client.Url.Path)
			if err != nil {
				return err
			}
			client.input = file
		} else {
			log.Printf("Connecting to %s\n", client.Url)
			getter := &http.Client {
				Timeout: client.Timeout,
			}
			response, err := getter.Get(client.Url.String())
			if err != nil {
				return err
			}
			client.socket = response
			client.input = response.Body
		}
		client.running = true
		log.Printf("Starting to pull from %s\n", client.Url)
		go client.pull()
		return nil
	}
	return ErrAlreadyConnected
}

// closes the connection
func (client *Client) Close() error {
	if client.input != nil {
		err := client.input.Close()
		client.input = nil
		return err
	}
	return ErrNoConnection
}

// returns the HTTP status code or 0 if not connected
func (client *Client) StatusCode() int {
	if client.socket != nil {
		return client.socket.StatusCode
	}
	// always return OK when our "socket" is an open file
	if client.input != nil {
		return http.StatusOK
	}
	return 0
}

// returns the HTTP status message or the empty string if not connected
func (client *Client) Status() string {
	return http.StatusText(client.StatusCode())
}

// returns true if the socket is connected
func (client *Client) Connected() bool {
	return client.running
}

// returns true if the connection is dead
// (i.e. has been opened and closed)
func (client *Client) Dead() bool {
	return !client.running && client.input != nil
}

// streams data from the socket to the queue
func (client *Client) pull() {
	log.Printf("Reading stream from %s\n", client.Url)
	
	for client.running {
		packet, err := ReadPacket(client.input)
		//log.Printf("Packet read complete, packet=%p, err=%p\n", packet, err)
		if err != nil {
			log.Printf("Got error on stream %s: %s\n", client.Url, err)
			client.running = false
		} else {
			if packet != nil {
				//log.Printf("Got a packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				//log.Printf("Got a packet (length %d)\n", len(packet))
				client.queue<- packet
			} else {
				log.Printf("No packet received\n")
			}
		}
	}
	
	log.Printf("Socket for stream %s closed\n", client.Url)
	client.Close()
	
	// reconnect after a while
	time.Sleep(10 * time.Second)
	go client.Connect()
}
