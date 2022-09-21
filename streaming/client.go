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

package streaming

import (
	"errors"
	"fmt"
	"github.com/onitake/restreamer/metrics"
	"github.com/onitake/restreamer/protocol"
	"github.com/onitake/restreamer/util"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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

var (
	metricSourceConnected = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "streaming_source_connected",
			Help: "Connection status, 0=disconnected 1=connected.",
		},
		[]string{"stream", "url"},
	)
	metricPacketsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "streaming_packets_received",
			Help: "Total number of MPEG-TS packets received.",
		},
		[]string{"stream", "url"},
	)
	metricBytesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "streaming_bytes_received",
			Help: "Total number of bytes received.",
		},
		[]string{"stream", "url"},
	)
)

func init() {
	metrics.MustRegister(metricSourceConnected)
	metrics.MustRegister(metricPacketsReceived)
	metrics.MustRegister(metricBytesReceived)
}

// connectCloser represents types that have a Connect() and a Close() method.
// It extends on the io.Closer type.
type connectCloser interface {
	// support the Closer interface
	io.Closer
	// and the Connect() method
	Connect() error
}

// dummyConnectCloser is a no-action implementation of the connectCloser interface.
type dummyConnectCloser struct {
}

func (*dummyConnectCloser) Close() error {
	return nil
}
func (*dummyConnectCloser) Connect() error {
	return nil
}

// Client implements a streaming HTTP client with failover support.
//
// Logging specification:
//
//	{
//	  "time": 1234 | unix timestamp in UTCS,
//	  "module": "client",
//	  "event": "" | error or upstream-connect or upstream-disconnect or upstream-loss or upstream-timeout or upstream-offline or client-streaming or client-stopped,
//	  when event=error:
//	    "error": "error-name"
//	    "error-specific key": "error-specific data"
//	  when event=retry:
//	    "retry: "99999" | seconds until retry
//	  when event=upstream-*:
//	    "url": "http://upstream/url" | upstream stream URL,
//	  when event=client-*:
//	    "client": "1.2.3.4:12" | client ip:port,
//	}
type Client struct {
	// name is a unique name for this stream, only used for logging and metrics
	name string
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
	running util.AtomicBool
	// stats is the statistics collector for this client
	stats metrics.Collector
	// listener is a downstream object that can handle connect/disconnect notifications
	listener connectCloser
	// queueSize is the size of the input queue
	queueSize uint
	// interf denotes a specific network interface to create the connection on
	// currently only supported for multicast
	interf *net.Interface
	// readBufferSize is the size of the receive on UDP sockets.
	readBufferSize int
	// packetSize defines the size of individual datagram packets (UDP)
	packetSize int
	// promCounter allows enabling/disabling Prometheus packet metrics.
	promCounter bool
}

// NewClient constructs a new streaming HTTP client, without connecting the socket yet.
// You need to call Connect() to do that.
//
// After a connection has been closed, the client will attempt to reconnect after a
// configurable delay. This delay is cumulative; if a connection has been up for longer,
// a reconnect will be attempted immediately.
//
// Arguments:
//
//	name: a unique name for this streaming client, used for metrics and logging
//	uris: a list of upstream URIs, used in random order
//	queue: the outgoing packet queue
//	timeout: the connect timeout
//	reconnect: the minimal reconnect delay
//	readtimeout: the read timeout
//	qsize: the input queue size
//	intf: the network interface to create multicast connections on
//	bufferSize: the UDP socket receive buffer size
//	packetSize: the UDP packet size
func NewClient(name string, uris []string, streamer *Streamer, timeout uint, reconnect uint, readtimeout uint, qsize uint, intf string, bufferSize uint, packetSize uint) (*Client, error) {
	urls := make([]*url.URL, len(uris))
	count := 0
	for _, uri := range uris {
		parsed, err := url.Parse(uri)
		if err == nil {
			urls[count] = parsed
			count++
		} else {
			logger.Logkv(
				"event", eventClientError,
				"error", errorClientParse,
				"message", fmt.Sprintf("Error parsing URL %s: %s", uri, err),
			)
		}
	}
	if count < 1 {
		return nil, ErrNoUrl
	}
	var pintf *net.Interface
	if intf != "" {
		var err error
		pintf, err = net.InterfaceByName(intf)
		if err != nil {
			logger.Logkv(
				"event", eventClientError,
				"error", errorClientInterface,
				"message", fmt.Sprintf("Error parsing network interface %s: %s", intf, err),
			)
		}
	}
	// this timeout is only used for establishing connections
	toduration := time.Duration(timeout) * time.Second
	dialer := &net.Dialer{
		Timeout:   toduration,
		KeepAlive: 0,
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   toduration,
		ResponseHeaderTimeout: toduration,
		ExpectContinueTimeout: toduration,
	}
	client := Client{
		name:      name,
		connector: dialer,
		getter: &http.Client{
			Transport: transport,
		},
		urls:           urls,
		response:       nil,
		input:          nil,
		Wait:           time.Duration(reconnect) * time.Second,
		ReadTimeout:    time.Duration(readtimeout) * time.Second,
		streamer:       streamer,
		running:        util.AtomicFalse,
		stats:          &metrics.DummyCollector{},
		listener:       &dummyConnectCloser{},
		queueSize:      qsize,
		interf:         pintf,
		readBufferSize: int(bufferSize * protocol.MpegTsPacketSize),
		packetSize:     int(packetSize),
	}
	return &client, nil
}

// SetCollector assigns a stats collector.
func (client *Client) SetCollector(stats metrics.Collector) {
	client.stats = stats
}

// SetStateListener adds a listener that will be notified when the client
// connection is closed or reconnected.
//
// Only one listener is supported.
func (client *Client) SetStateListener(listener connectCloser) {
	client.listener = listener
}

// SetInhibit calls the SetInhibit function on the attached streamer.
func (client *Client) SetInhibit(inhibit bool) {
	// delegate to the streamer
	client.streamer.SetInhibit(inhibit)
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
	return util.LoadBool(&client.running)
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
				logger.Logkv(
					"event", eventClientRetry,
					"retry", wait.Seconds(),
					"message", fmt.Sprintf("Retrying after %0.0f seconds.", wait.Seconds()),
				)
				time.Sleep(wait)
			}
			// update the deadline
			deadline = time.Now().Add(client.Wait)
		}

		// pick the next server
		url := client.urls[next]
		next = (next + 1) % len(client.urls)

		// connect
		logger.Logkv(
			"event", eventClientConnecting,
			"url", url.String(),
		)
		err := client.start(url)
		if err != nil {
			// not handled, log
			logger.Logkv(
				"event", eventClientError,
				"error", errorClientConnect,
				"url", url.String(),
				"message", err.Error(),
			)
		}

		if client.Wait == 0 {
			logger.Logkv(
				"event", eventClientOffline,
				"url", url.String(),
				"message", "Reconnecting disabled. Stream will stay offline.",
			)
		}
	}
}

// start connects the socket, sends the HTTP request and starts streaming.
func (client *Client) start(urly *url.URL) error {
	/*client.logger.Logkv(
		"event", eventClientDebug,
		"debug", map[string]interface{}{
			"timeout": client.Timeout,
		},
		"urly": urly.String(),
	)*/
	if client.input == nil {
		switch urly.Scheme {
		// handled by os.Open
		case "file":
			logger.Logkv(
				"event", eventClientOpenPath,
				"path", urly.Path,
				"message", fmt.Sprintf("Opening %s.", urly.Path),
			)
			// prevent blocking on opening named pipes for reading.
			//
			// O_NONBLOCK is not portable, but should work at least on POSIX-compliant systems.
			// we'd still need to reset back to blocking I/O once the file is open.
			// O_RDWR without O_NONBLOCK will also work, according to POSIX semantics.
			// since we never write to the pipe, this shouldn't cause problems.
			// and non-POSIX systems probably don't support named pipes well anyway, so YMMV.
			//
			// see: https://pubs.opengroup.org/onlinepubs/007908799/xsh/open.html
			// and: https://pubs.opengroup.org/onlinepubs/9699919799/functions/write.html
			//file, err := os.OpenFile(urly.Path, syscall.O_RDONLY | syscall.O_NONBLOCK, 0666)
			//syscall.SetNonblock(file.Fd(), false)
			file, err := os.OpenFile(urly.Path, os.O_RDWR, 0666)
			if err != nil {
				return err
			}
			client.input = file
		// both handled by http.Client
		case "http":
			fallthrough
		case "https":
			logger.Logkv(
				"event", eventClientOpenHttp,
				"urly", urly.String(),
				"message", fmt.Sprintf("Connecting to %s.", urly),
			)
			request, err := http.NewRequest("GET", urly.String(), nil)
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
			logger.Logkv(
				"event", eventClientOpenTcp,
				"host", urly.Host,
				"message", fmt.Sprintf("Connecting TCP socket to %s.", urly.Host),
			)
			conn, err := client.connector.Dial(urly.Scheme, urly.Host)
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
			logger.Logkv(
				"event", eventClientOpenDomain,
				"path", urly.Path,
				"message", fmt.Sprintf("Connecting domain socket to %s.", urly.Path),
			)
			conn, err := client.connector.Dial(urly.Scheme, urly.Path)
			if err != nil {
				return err
			}
			client.input = conn
		case "udp":
			addr, err := net.ResolveUDPAddr("udp", urly.Host)
			if err != nil {
				return err
			}
			var conn *net.UDPConn
			if addr.IP.IsMulticast() {
				logger.Logkv(
					"event", eventClientOpenUdpMulticast,
					"address", addr,
					"message", fmt.Sprintf("Joining UDP multicast group %s on interface %v.", urly.Host, client.interf),
				)
				var err error
				conn, err = net.ListenMulticastUDP("udp", client.interf, addr)
				if err != nil {
					return err
				}
			} else {
				logger.Logkv(
					"event", eventClientOpenUdp,
					"address", addr,
					"message", fmt.Sprintf("Connecting to UDP address %s.", addr),
				)
				var err error
				conn, err = net.ListenUDP("udp", addr)
				if err != nil {
					return err
				}
			}
			if err := conn.SetReadBuffer(client.readBufferSize); err != nil {
				logger.Logkv(
					"event", eventClientError,
					"error", errorClientSetBufferSize,
					"address", addr,
					"message", fmt.Sprintf("Error setting read buffer size: %v (ignored)", err),
				)
			}
			client.input = protocol.NewFixedReader(conn, client.packetSize)
		case "fork":
			command := urly.Hostname()
			arguments, err := url.QueryUnescape(urly.RawQuery)
			if err != nil {
				return err
			}
			logger.Logkv(
				"event", eventClientOpenFork,
				"command", command,
				"arguments", arguments,
				"message", fmt.Sprintf("Executing command source: %s %s", command, arguments),
			)
			// FIXME This assumes none of the command line arguments contain spaces.
			// To support arbitrary command lines and, in particular, shell commands, we need to find a different way
			// to separate individual arguments. For example, we could use a query list with the arguments as
			// keys and empty values. Or, we could simply use a "arg" key and specify it multiple times.
			// url.Values object is a multimap, after all.
			arglist := strings.Split(arguments, " ")
			cmd, err := protocol.NewForkReader(command, arglist)
			if err != nil {
				return err
			}
			client.input = cmd
		default:
			return ErrInvalidProtocol
		}

		// start streaming
		util.StoreBool(&client.running, true)
		logger.Logkv(
			"event", eventClientPull,
			"urly", urly.String(),
			"message", fmt.Sprintf("Starting to pull stream %s.", urly),
		)
		err := client.pull(urly)
		logger.Logkv(
			"event", eventClientClosed,
			"urly", urly.String(),
			"message", fmt.Sprintf("Socket for stream %s closed", urly),
		)

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
	var queue chan protocol.MpegTsPacket
	// save a few bytes
	var packet protocol.MpegTsPacket

	for util.LoadBool(&client.running) {
		// somewhat hacky read timeout:
		// close the connection when the timer fires.
		// we need this because the Go I/O implementation does not support
		// deadlines on reads or writes.
		var timer *time.Timer
		if client.ReadTimeout > 0 {
			timer = time.AfterFunc(client.ReadTimeout, func() {
				logger.Logkv(
					"event", eventClientReadTimeout,
					"message", "Read timeout exceeded, closing connection",
				)
				client.input.Close()
			})
		}
		// read a packet
		//log.Printf("Reading a packet from %p\n", client.input)
		packet, err = protocol.ReadMpegTsPacket(client.input)
		// we got a packet, stop the timer and drain it
		if timer != nil && !timer.Stop() {
			logger.Logkv(
				"event", eventClientTimerStop,
				"url", url.String(),
				"message", fmt.Sprintf("Stopping timer on %s", url),
			)
			select {
			case <-timer.C:
			default:
			}
			logger.Logkv(
				"event", eventClientTimerStopped,
				"url", url.String(),
				"message", fmt.Sprintf("Stopped timer on %s", url),
			)
		}
		//log.Printf("Packet read complete, packet=%p, err=%p\n", packet, err)
		if err != nil {
			util.StoreBool(&client.running, false)
		} else {
			if packet != nil {
				// report connection up
				if queue == nil {
					client.listener.Connect()
					client.stats.SourceConnected()
					metricSourceConnected.With(prometheus.Labels{"stream": client.name, "url": url.String()}).Set(1.0)
					logger.Logkv(
						"event", eventClientStarted,
						"url", url.String(),
					)
					queue = make(chan protocol.MpegTsPacket, client.queueSize)
					go client.streamer.Stream(queue)
				}

				// report the packet
				client.stats.PacketReceived()
				if client.promCounter {
					metricPacketsReceived.With(prometheus.Labels{"stream": client.name, "url": url.String()}).Inc()
					metricBytesReceived.With(prometheus.Labels{"stream": client.name, "url": url.String()}).Add(protocol.MpegTsPacketSize)
				}

				//log.Printf("Got a packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				//log.Printf("Got a packet (length %d)\n", len(packet))
				queue <- packet
			} else {
				logger.Logkv(
					"event", eventClientNoPacket,
					"url", url.String(),
					"message", "No packet received",
				)
			}
		}
	}

	// and the connection is gone
	if queue != nil {
		logger.Logkv(
			"event", eventClientTimerKill,
			"url", url.String(),
			"message", fmt.Sprintf("Killing queue on %s", url),
		)
		close(queue)
		client.stats.SourceDisconnected()
		metricSourceConnected.With(prometheus.Labels{"stream": client.name, "url": url.String()}).Set(0.0)
		logger.Logkv(
			"event", eventClientStopped,
			"url", url.String(),
		)
	}

	return err
}
