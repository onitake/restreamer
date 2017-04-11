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
	Urls []*url.URL
	// the response, including the body reader
	response *http.Response
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
	toduration := time.Duration(timeout) * time.Second
	dialer := &net.Dialer{
		Timeout: toduration,
		KeepAlive: 0,
		DualStack: true,
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: dialer.DialContext,
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
		Urls: urls,
		response: nil,
		input: nil,
		Timeout: toduration,
		Wait: time.Duration(reconnect) * time.Second,
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

// Close closes the active outgoing connection.
// The connection is kept running, so a new connection may be established right away.
func (client *Client) Close() error {
	if client.input != nil {
		err := client.input.Close()
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
		next := randomizer.Intn(len(client.Urls))
		url := client.Urls[next]
		
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
		
		// start streaming
		client.running = true
		log.Printf("Starting to pull stream %s\n", url)
		err := client.pull()
		log.Printf("Socket for stream %s closed\n", url)
		
		return err
	}
	return ErrAlreadyConnected
}

// pull streams data from the socket into the queue.
func (client *Client) pull() error {
	var err error
	// will be set as soon as the first packet has been received
	// necessary, because we only report connections as online that actually send data.
	connected := false
	
	var packet Packet
	for client.running {
		packet, err = ReadPacket(client.input)
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
	
	client.Close()
	client.input = nil
	client.response = nil
	
	return err
}
