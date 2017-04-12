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
	"math/rand"
)

const (
	eventClientDebug = "debug"
	eventClientError = "error"
	eventClientRetry = "retry"
	eventClientConnecting = "connecting"
	eventClientConnectionLoss = "loss"
	eventClientConnectTimeout = "timeout"
	eventClientOffline = "offline"
	eventClientStarted = "started"
	eventClientStopped = "stopped"
	//
	errorClientConnect = "connect"
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
	ErrNoUrl = errors.New("restreamer: no parseable upstream URL")
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
//
// Logging specification:
// {
//   "time": 1234 | unix timestamp in UTCS,
//   "module": "client",
//   "event": "" | error or upstream-connect or upstream-disconnect or upstream-loss or upstream-timeout or upstream-offline or client-streaming or client-stopped,
//   when event=error:
//     "error": "error-name"
//     "error-specific key": "error-specific data"
//   when event=retry:
//     "retry: "99999" | seconds until retry
//   when event=upstream-*:
//     "url": "http://upstream/url" | upstream stream URL,
//   when event=client-*:
//     "client": "1.2.3.4:12" | client ip:port,
// }
	
type Client struct {
	// a network dialer for TCP, UDP and HTTP
	connector *net.Dialer
	// a generic HTTP client
	getter *http.Client
	// the URLs to GET (either of them)
	urls []*url.URL
	// the response, including the body reader
	response *http.Response
	// the input stream (socket)
	input io.ReadCloser
	// Wait is the time before reconnecting a disconnected upstream.
	// This is a deadline: If a connection (or connection attempt) takes longer
	// than this duration, a reconnection is attempted immediately.
	Wait time.Duration
	// ReadTimeout is the timeout for individual packet reads
	ReadTimeout time.Duration
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
//
// After a connection has been closed, the client will attempt to reconnect after a
// configurable delay. This delay is cumulative; if a connection has been up for longer,
// a reconnect will be attempted immediately.
//
// Arguments:
//   uris: a list of upstream URIs, used in random order
//   queue: the outgoing packet queue
//   timeout: the connect timeout
//   reconnect: the minimal reconnect delay
//   readtimeout: the read timeout
func NewClient(uris []string, queue chan<- Packet, timeout uint, reconnect uint, readtimeout uint) (*Client, error) {
	urls := make([]*url.URL, len(uris))
	count := 0
	for _, uri := range uris {
		parsed, err := url.Parse(uri)
		if err == nil {
			urls[count] = parsed
			count++
		} else {
			log.Printf("Error parsing URL %s: %s", uri, err)
		}
	}
	if count < 1 {
		return nil, ErrNoUrl
	}
	// this timeout is only used for establishing connections
	toduration := time.Duration(timeout) * time.Second
	dialer := &net.Dialer{
		Timeout: toduration,
		KeepAlive: 0,
		DualStack: true,
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: dialer.Dial,
		DisableKeepAlives: true,
		TLSHandshakeTimeout: toduration,
		ResponseHeaderTimeout: toduration,
		ExpectContinueTimeout: toduration,
	}
	client := Client {
		connector: dialer,
		getter: &http.Client{
			Transport: transport,
		},
		urls: urls,
		response: nil,
		input: nil,
		Wait: time.Duration(reconnect) * time.Second,
		ReadTimeout: time.Duration(readtimeout) * time.Second,
		queue: queue,
		running: false,
		stats: &DummyCollector{},
		logger: &DummyLogger{},
		listener: &DummyConnectCloser{},
	}
	return &client, nil
}

// SetLogger assigns a logger.
func (client *Client) SetLogger(logger JsonLogger) {
	client.logger = &ModuleLogger{
		Logger: logger,
		Defaults: map[string]interface{} {
			"module": "client",
		},
	}
}

// SetCollector assigns a stats collector.
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

// Close closes the active upstream connection.
//
// This will cause the streaming thread to fail and try to reestablish
// a connection (unless reconnects are disabled).
func (client *Client) Close() error {
	if client.input != nil {
		err := client.input.Close()
		return err
	}
	return ErrNoConnection
}

// Connect spawns the connection loop.
//
// Do not call this method multiple times!
func (client *Client) Connect() {
	go client.loop()
}

// StatusCode returns the HTTP status code, or 0 if not connected.
func (client *Client) StatusCode() int {
	if client.response != nil {
		return client.response.StatusCode
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
	
	// connect order randomizer
	randomizer := rand.New(rand.NewSource(time.Now().Unix()))

	// deadline to avoid a busy loop, but still allow an immediate reconnect on loss
	deadline := time.Now().Add(client.Wait)
	
	for first || client.Wait != 0 {
		if first {
			// there is only one first attempt
			first = false
		} else {
			// sleep if this is not the first attempt,
			// but sleep only if the deadline has not been reached yet
			now := time.Now()
			if now.Before(deadline) {
				wait := deadline.Sub(now)
				// first attempt, let's see if it works!
				client.logger.Log(Dict{
					"event": eventClientRetry,
					"retry": wait.Seconds(),
				})
				log.Printf("Retrying after %0.0f seconds.\n", wait.Seconds());
				time.Sleep(wait)
			}
			// update the deadline
			deadline = time.Now().Add(client.Wait)
		}
		
		// pick a random server
		next := randomizer.Intn(len(client.urls))
		url := client.urls[next]
		
		// connect
		client.logger.Log(Dict{
			"event": eventClientConnecting,
			"url": url.String(),
		})
		err := client.start(url)
		if err != nil {
			// not handled, log
			client.logger.Log(Dict{
				"event": eventClientError,
				"error": errorClientConnect,
				"url": url.String(),
				"message": err.Error(),
			})
			log.Printf("Got error on stream %s: %s\n", url, err)
		}
		
		if client.Wait == 0 {
			client.logger.Log(Dict{
				"event": eventClientOffline,
				"url": url.String(),
			})
			log.Print("Reconnecting disabled. Stream will stay offline.\n");
		}
	}
}

// start connects the socket, sends the HTTP request and starts streaming.
func (client *Client) start(url *url.URL) error {
	/*client.logger.Log(Dict{
		"event": eventClientDebug,
		"debug": Dict{
			"timeout": client.Timeout,
		},
		"url": url.String(),
	})*/
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
			request, err := http.NewRequest("GET", url.String(), nil)
			if err != nil {
				return err
			}
			response, err := client.getter.Do(request)
			if err != nil {
				return err
			}
			client.response = response
			client.input = response.Body
		// handled directly by net.Dialer
		case "tcp":
			log.Printf("Connecting TCP socket to %s\n", url.Host)
			conn, err := client.connector.Dial(url.Scheme, url.Host)
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
			conn, err := client.connector.Dial(url.Scheme, url.Path)
			if err != nil {
				return err
			}
			client.input = conn
		default:
			return ErrInvalidProtocol
		}
		
		// start streaming
		client.running = true
		log.Printf("Starting to pull stream %s\n", url)
		err := client.pull(url)
		log.Printf("Socket for stream %s closed\n", url)
		
		// cleanup
		client.Close()
		client.input = nil
		client.response = nil
		
		return err
	}
	return ErrAlreadyConnected
}

// pull streams data from the socket into the queue.
func (client *Client) pull(url *url.URL) error {
	// declare here so we can send back individual errors
	var err error
	// will be set as soon as the first packet has been received
	// necessary, because we only report connections as online that actually send data.
	connected := false
	
	var packet Packet
	for client.running {
		// somewhat hacky read timeout:
		// close the connection when the timer fires.
		var timer *time.Timer
		if client.ReadTimeout > 0 {
			timer = time.AfterFunc(client.ReadTimeout, func() {
				log.Printf("Read timeout exceeded, closing connection\n")
				client.input.Close()
			})
		}
		// read a packet
		packet, err = ReadPacket(client.input)
		// we got a packet, stop the timer (and drain it)
		if timer != nil && !timer.Stop() {
			<-timer.C
		}
		//log.Printf("Packet read complete, packet=%p, err=%p\n", packet, err)
		if err != nil {
			client.running = false
		} else {
			if packet != nil {
				// report connection up
				if !connected {
					client.listener.Connect()
					client.stats.SourceConnected()
					client.logger.Log(Dict{
						"event": eventClientStarted,
						"url": url.String(),
					})
					connected = true
				}
				
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
	if connected {
		client.listener.Close()
		client.stats.SourceDisconnected()
		client.logger.Log(Dict{
			"event": eventClientStopped,
			"url": url.String(),
		})
	}
	
	return err
}
