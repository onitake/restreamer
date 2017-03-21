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
	// ErrNoUrl is thrown when the list of upstream URLs was empty
	ErrNoUrl = errors.New("restreamer: no upstream URL")
)

// ConnectCloser is an interface for objects that support a Connect() and Close() method.
// Most suitable for self-managed classes that have an 'offline' and 'online' state.
type ConnectCloser interface {
	// support the Closer interface
	io.Closer
	// and the Connect() method
	Connect() error
}

// DummyConnectCloser is a no-action implementation of the ConnectCloser interface.
type DummyConnectCloser struct {
}
func (*DummyConnectCloser) Close() error {
	return nil
}
func (*DummyConnectCloser) Connect() error {
	return nil
}

// Client implements a streaming HTTP client with failover support.
type Client struct {
	// the URLs to GET (either of them)
	Urls []*url.URL
	// the response, including the body reader
	socket *http.Response
	// the input stream (socket)
	input io.ReadCloser
	// the I/O timeout
	Timeout time.Duration
	// Wait time before reconnecting a disconnected upstream.
	// This is a deadline: If a connection (or connection attempt) takes longer
	// than this duration, a reconnection is attempted immediately.
	Wait time.Duration
	// the packet queue
	queue chan<- Packet
	// true while the client is streaming into the queue
	// TODO make this atomic
	running bool
	// the stats collector for this client
	stats Collector
	// a json logger
	logger JsonLogger
	// a downstream object that can handle connect/disconnect notifications
	listener ConnectCloser
}

// NewClient constructs a new streaming HTTP client, without connecting the socket yet.
// You need to call Connect() to do that.
// After a connection has been closed, the client will attempt to reconnect after a configurable delay.
func NewClient(uris []string, queue chan<- Packet, timeout uint, reconnect uint) (*Client, error) {
	if len(uris) < 1 {
		return nil, ErrNoUrl
	}
	urls := make([]*url.URL, len(uris))
	for i, uri := range uris {
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		urls[i] = parsed
	}
	return &Client {
		Urls: urls,
		socket: nil,
		input: nil,
		Timeout: time.Duration(timeout) * time.Second,
		Wait: time.Duration(reconnect) * time.Second,
		queue: queue,
		running: false,
		stats: &DummyCollector{},
		logger: &DummyLogger{},
		listener: &DummyConnectCloser{},
	}, nil
}

// Assigns a logger
func (client *Client) SetLogger(logger JsonLogger) {
	client.logger = logger
}

// Assigns a stats collector
func (client *Client) SetCollector(stats Collector) {
	client.stats = stats
}

// SetStateListener adds a listener that will be notified when the client
// connection is closed or reconnected.
//
// Only one listener is supported.
func (client *Client) SetStateListener(listener ConnectCloser) {
	client.listener = listener
}

// Closes the connection.
func (client *Client) Close() error {
	if client.input != nil {
		err := client.input.Close()
		client.input = nil
		return err
	}
	return ErrNoConnection
}

// Connect spawns the connection loop.
func (client *Client) Connect() {
	go client.loop()
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

// loop tries to connect and loops until successful.
// If client.Wait is 0, it only tries once.
func (client *Client) loop() {
	first := true
	
	// deadline to avoid a busy loop, but still allow an immediate reconnect on loss
	deadline := time.Now().Add(client.Wait)
	
	for first || client.Wait != 0 {
		// sleep if this is not the first attempt
		if !first {
			// sleep only if the deadline has not been reached yet
			now := time.Now()
			if now.Before(deadline) {
				wait := deadline.Sub(now)
				log.Printf("Retrying after %0.0f seconds.\n", wait.Seconds());
				time.Sleep(wait)
			}
			// update the deadline
			deadline = time.Now().Add(client.Wait)
		} else {
			// there is only one first attempt
			first = false
		}
		
		// and try each upstream, in order
		// TODO use random order
		for _, url := range client.Urls {
			err := client.start(url)
			if err == nil {
				// connection handled, out
				break
			} else {
				// not handled, print and try next
				log.Printf("Got error on stream %s: %s\n", url, err)
			}
		}
		
		if client.Wait == 0 {
			log.Print("Reconnecting disabled. Stream will stay offline.\n");
		}
	}
}

// start connects the socket, sends the HTTP request and starts streaming.
func (client *Client) start(url *url.URL) error {
	if client.input == nil {
		switch url.Scheme {
		// handled by os.Open
		case "file":
			log.Printf("Opening %s\n", url.Path)
			file, err := os.Open(url.Path)
			if err != nil {
				return err
			}
			client.input = file
		// both handled by http.Client
		case "http":
			fallthrough
		case "https":
			log.Printf("Connecting to %s\n", url)
			getter := &http.Client {
				Timeout: client.Timeout,
			}
			response, err := getter.Get(url.String())
			if err != nil {
				return err
			}
			client.socket = response
			client.input = response.Body
		// handled directly by net.Dialer
		case "tcp":
			log.Printf("Connecting TCP socket to %s\n", url.Host)
			dialer := &net.Dialer {
				Timeout: client.Timeout,
			}
			conn, err := dialer.Dial(url.Scheme, url.Host)
			if err != nil {
				return err
			}
			client.input = conn
		// handled by net.Dialer too, but different URL semantics
		case "unix":
			fallthrough
		case "unixgram":
			fallthrough
		case "unixpacket":
			log.Printf("Connecting domain socket to %s\n", url.Path)
			dialer := &net.Dialer {
				Timeout: client.Timeout,
			}
			conn, err := dialer.Dial(url.Scheme, url.Path)
			if err != nil {
				return err
			}
			client.input = conn
		default:
			return ErrInvalidProtocol
		}
		
		// we're connected now
		client.listener.Connect()
		
		// start streaming
		client.running = true
		log.Printf("Starting to pull stream %s\n", url)
		err := client.pull()
		log.Printf("Socket for stream %s closed\n", url)
		
		// and that's it
		client.listener.Close()
		
		return err
	}
	return ErrAlreadyConnected
}

// pull streams data from the socket into the queue.
func (client *Client) pull() error {
	var err error
	
	// we're connected now
	client.stats.SourceConnected()
	
	var packet Packet
	for client.running {
		packet, err = ReadPacket(client.input)
		//log.Printf("Packet read complete, packet=%p, err=%p\n", packet, err)
		if err != nil {
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
	
	client.Close()
	
	return err
}
