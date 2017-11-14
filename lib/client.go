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
	"fmt"
	"time"
	"errors"
	"net"
	"net/http"
	"net/url"
)

const (
	moduleClient = "client"
	//
	eventClientDebug = "debug"
	eventClientError = "error"
	eventClientRetry = "retry"
	eventClientConnecting = "connecting"
	eventClientConnectionLoss = "loss"
	eventClientConnectTimeout = "connect_timeout"
	eventClientOffline = "offline"
	eventClientStarted = "started"
	eventClientStopped = "stopped"
	eventClientOpenPath = "open_path"
	eventClientOpenHttp = "open_http"
	eventClientOpenTcp = "open_tcp"
	eventClientOpenDomain = "open_domain"
	eventClientPull = "pull"
	eventClientClosed = "closed"
	eventClientTimerStop = "timer_stop"
	eventClientTimerStopped = "timer_stopped"
	eventClientNoPacket = "nopacket"
	eventClientTimerKill = "killed"
	eventClientReadTimeout = "read_timeout"
	//
	errorClientConnect = "connect"
	errorClientParse = "parse"
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
	// connector is a network dialer for TCP, UDP and HTTP
	connector *net.Dialer
	// getter is a generic HTTP client
	getter *http.Client
	// urls is the URLs to GET (either of them)
	urls []*url.URL
	// response is the HTTP response, including the body reader
	response *http.Response
	// input is the input stream (socket)
	input io.ReadCloser
	// Wait is the time before reconnecting a disconnected upstream.
	// This is a deadline: If a connection (or connection attempt) takes longer
	// than this duration, a reconnection is attempted immediately.
	Wait time.Duration
	// ReadTimeout is the timeout for individual packet reads
	ReadTimeout time.Duration
	// streamer is the attached packet distributor
	streamer *Streamer
	// running is true while the client is streaming into the queue.
	// Use ReadBool(client.running) to get the current value.
	running AtomicBool
	// stats is the statistics collector for this client
	stats Collector
	// logger is our augmented logging facility
	logger *ModuleLogger
	// listener is a downstream object that can handle connect/disconnect notifications
	listener ConnectCloser
	// queueSize is the size of the input queue
	queueSize uint
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
//   qsize: the input queue size
func NewClient(uris []string, streamer *Streamer, timeout uint, reconnect uint, readtimeout uint, qsize uint) (*Client, error) {
	logger := &ModuleLogger{
		Logger: &ConsoleLogger{},
		Defaults: Dict{
			"module": moduleClient,
		},
		AddTimestamp: true,
	}
	urls := make([]*url.URL, len(uris))
	count := 0
	for _, uri := range uris {
		parsed, err := url.Parse(uri)
		if err == nil {
			urls[count] = parsed
			count++
		} else {
			logger.Log(Dict{
				"event": eventClientError,
				"error": errorClientParse,
				"message": fmt.Sprintf("Error parsing URL %s: %s", uri, err),
			})
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
		streamer: streamer,
		running: AtomicFalse,
		stats: &DummyCollector{},
		logger: logger,
		listener: &DummyConnectCloser{},
		queueSize: qsize,
	}
	return &client, nil
}

// SetLogger assigns a backing logger, while keeping the current module defaults.
func (client *Client) SetLogger(logger JsonLogger) {
	client.logger.Logger = logger
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
	return LoadBool(&client.running)
}

// loop tries to connect and loops until successful.
// If client.Wait is 0, it only tries once.
func (client *Client) loop() {
	first := true
	
	// deadline to avoid a busy loop, but still allow an immediate reconnect on loss
	deadline := time.Now().Add(client.Wait)
	
	next := 0
	
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
				client.logger.Log(Dict{
					"event": eventClientRetry,
					"retry": wait.Seconds(),
					"message": fmt.Sprintf("Retrying after %0.0f seconds.", wait.Seconds()),
				})
				time.Sleep(wait)
			}
			// update the deadline
			deadline = time.Now().Add(client.Wait)
		}
		
		// pick the next server
		url := client.urls[next]
		next = (next + 1) % len(client.urls)
		
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
		}
		
		if client.Wait == 0 {
			client.logger.Log(Dict{
				"event": eventClientOffline,
				"url": url.String(),
				"message": "Reconnecting disabled. Stream will stay offline.",
			})
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
			client.logger.Log(Dict{
				"event": eventClientOpenPath,
				"path": url.Path,
				"message": fmt.Sprintf("Opening %s.", url.Path),
			})
			file, err := os.Open(url.Path)
			if err != nil {
				return err
			}
			client.input = file
		// both handled by http.Client
		case "http":
			fallthrough
		case "https":
			client.logger.Log(Dict{
				"event": eventClientOpenHttp,
				"url": url.String(),
				"message": fmt.Sprintf("Connecting to %s.", url),
			})
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
			client.logger.Log(Dict{
				"event": eventClientOpenTcp,
				"host": url.Host,
				"message": fmt.Sprintf("Connecting TCP socket to %s.", url.Host),
			})
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
			client.logger.Log(Dict{
				"event": eventClientOpenDomain,
				"path": url.Path,
				"message": fmt.Sprintf("Connecting domain socket to %s.", url.Path),
			})
			conn, err := client.connector.Dial(url.Scheme, url.Path)
			if err != nil {
				return err
			}
			client.input = conn
		default:
			return ErrInvalidProtocol
		}
		
		// start streaming
		StoreBool(&client.running, true)
		client.logger.Log(Dict{
			"event": eventClientPull,
			"url": url.String(),
			"message": fmt.Sprintf("Starting to pull stream %s.", url),
		})
		err := client.pull(url)
		client.logger.Log(Dict{
			"event": eventClientClosed,
			"url": url.String(),
			"message": fmt.Sprintf("Socket for stream %s closed", url),
		})
		
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
	// the packet queue will be allocated and connected to the streamer as soon as the first packet has been received
	var queue chan Packet
	// save a few bytes
	var packet Packet
	
	for LoadBool(&client.running) {
		// somewhat hacky read timeout:
		// close the connection when the timer fires.
		var timer *time.Timer
		if client.ReadTimeout > 0 {
			timer = time.AfterFunc(client.ReadTimeout, func() {
				client.logger.Log(Dict{
					"event": eventClientReadTimeout,
					"message": "Read timeout exceeded, closing connection",
				})
				client.input.Close()
			})
		}
		// read a packet
		//log.Printf("Reading a packet from %p\n", client.input)
		packet, err = ReadPacket(client.input)
		// we got a packet, stop the timer and drain it
		if timer != nil && !timer.Stop() {
			client.logger.Log(Dict{
				"event": eventClientTimerStop,
				"url": url.String(),
				"message": fmt.Sprintf("Stopping timer on %s", url),
			})
			select {
				case <-timer.C:
				default:
			}
			client.logger.Log(Dict{
				"event": eventClientTimerStopped,
				"url": url.String(),
				"message": fmt.Sprintf("Stopped timer on %s", url),
			})
		}
		//log.Printf("Packet read complete, packet=%p, err=%p\n", packet, err)
		if err != nil {
			StoreBool(&client.running, false)
		} else {
			if packet != nil {
				// report connection up
				if queue == nil {
					client.listener.Connect()
					client.stats.SourceConnected()
					client.logger.Log(Dict{
						"event": eventClientStarted,
						"url": url.String(),
					})
					queue = make(chan Packet, client.queueSize)
					go client.streamer.Stream(queue)
				}
				
				// report the packet
				client.stats.PacketReceived()
				
				//log.Printf("Got a packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				//log.Printf("Got a packet (length %d)\n", len(packet))
				queue<- packet
			} else {
				client.logger.Log(Dict{
					"event": eventClientNoPacket,
					"url": url.String(),
					"message": "No packet received",
				})
			}
		}
	}
	
	// and the connection is gone
	if queue != nil {
		client.logger.Log(Dict{
			"event": eventClientTimerKill,
			"url": url.String(),
			"message": fmt.Sprintf("Killing queue on %s", url),
		})
		close(queue)
		client.stats.SourceDisconnected()
		client.logger.Log(Dict{
			"event": eventClientStopped,
			"url": url.String(),
		})
	}
	
	return err
}
