/* Copyright (c) 2016-2017 Gregor Riepl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package restreamer

import (
	"os"
	"io"
	"log"
	"time"
	"errors"
	"net"
	"net/http"
	"net/url"
)

const (
	DefaultTimeout time.Duration = 10 * time.Second
)

var (
	// ErrInvalidProtocol is thrown when an invalid protocol was specified.
	// See the docs and example config for a list of supported protocols.
	ErrInvalidProtocol = errors.New("restreamer: unsupported protocol")
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

// Client implements a streaming HTTP client
type Client struct {
	// the URL to GET
	Url *url.URL
	// the response, including the body reader
	socket *http.Response
	// the input stream (socket)
	input io.ReadCloser
	// the I/O timeout
	Timeout time.Duration
	// wait time before reconnecting a disconnected upstream
	Wait time.Duration
	// the packet queue
	queue chan<- Packet
	// the stats collector for this stream
	stats Collector
	// true while the client is streaming into the queue
	// TODO make this atomic
	running bool
}

// NewClient constructs a new streaming HTTP client, without connecting the socket yet.
// You need to call Connect() to do that.
// After a connection has been closed, the client will attempt to reconnect after a configurable delay.
func NewClient(uri string, queue chan<- Packet, timeout uint, reconnect uint, stats Collector) (*Client, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return &Client {
		Url: parsed,
		socket: nil,
		input: nil,
		Timeout: time.Duration(timeout) * time.Second,
		Wait: time.Duration(reconnect) * time.Second,
		queue: queue,
		stats: stats,
		running: false,
	}, nil
}

// Connect spawns the connection loop.
func (client *Client) Connect() {
	go client.loop()
}

// loop tries to connect and loops until successful.
// If client.Wait is 0, it only tries once.
func (client *Client) loop() {
	first := true
	
	for first || client.Wait != 0 {
		// sleep if this is not the first attempt
		if !first {
			time.Sleep(client.Wait)
		} else {
			// there is only one first attempt
			first = false
		}
		
		// and connect
		err := client.start()
		if err != nil {
			log.Print(err)
		}
		
		if client.Wait != 0 {
			log.Printf("Retrying after %0.0f seconds.\n", client.Wait.Seconds());
		} else {
			log.Print("Reconnecting disabled. Stream will stay offline.\n");
		}
	}
}

// start connects the socket, sends the HTTP request and starts streaming.
func (client *Client) start() error {
	if client.input == nil {
		switch client.Url.Scheme {
		// handled by os.Open
		case "file":
			log.Printf("Opening %s\n", client.Url.Path)
			file, err := os.Open(client.Url.Path)
			if err != nil {
				return err
			}
			client.input = file
		// both handled by http.Client
		case "http":
			fallthrough
		case "https":
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
		// handled by 
		case "tcp":
			addr, err := net.ResolveTCPAddr("tcp", client.Url.Host)
			if err != nil {
				return err
			}
			log.Printf("Connecting TCP socket to %s:%d\n", addr.IP, addr.Port)
			conn, err := net.DialTCP("tcp", nil, addr)
			if err != nil {
				return err
			}
			client.input = conn
		default:
			return ErrInvalidProtocol
		}
		
		client.running = true
		log.Printf("Starting to pull from %s\n", client.Url)
		client.pull()
		
		return nil
	}
	return ErrAlreadyConnected
}

// Close closes the connection.
func (client *Client) Close() error {
	if client.input != nil {
		err := client.input.Close()
		client.input = nil
		return err
	}
	return ErrNoConnection
}

// StatusCode returns the HTTP status code, or 0 if not connected.
func (client *Client) StatusCode() int {
	if client.socket != nil {
		return client.socket.StatusCode
	}
	// other protocols don't have status codes, so just return 200 if connected
	if client.input != nil {
		return http.StatusOK
	}
	return 0
}

// Status returns the HTTP status message, or the empty string if not connected.
func (client *Client) Status() string {
	return http.StatusText(client.StatusCode())
}

// Connected returns true if the socket is connected.
func (client *Client) Connected() bool {
	return client.running
}

// pull streams data from the socket into the queue.
func (client *Client) pull() {
	log.Printf("Reading stream from %s\n", client.Url)
	
	// we're connected now
	client.stats.SourceConnected()
	
	for client.running {
		packet, err := ReadPacket(client.input)
		//log.Printf("Packet read complete, packet=%p, err=%p\n", packet, err)
		if err != nil {
			log.Printf("Got error on stream %s: %s\n", client.Url, err)
			client.running = false
		} else {
			if packet != nil {
				// report the packet
				client.stats.PacketReceived()
				
				//log.Printf("Got a packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				//log.Printf("Got a packet (length %d)\n", len(packet))
				client.queue<- packet
			} else {
				log.Printf("No packet received\n")
			}
		}
	}
	
	// and the connection is gone
	client.stats.SourceDisconnected()
	
	log.Printf("Socket for stream %s closed\n", client.Url)
	client.Close()
}
