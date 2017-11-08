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
	"fmt"
	"sync"
	"errors"
	"net/http"
)

const (
	moduleStreamer = "streamer"
	//
	eventStreamerError = "error"
	eventStreamerQueueStart = "queuestart"
	eventStreamerStart = "start"
	eventStreamerStop = "stop"
	eventStreamerClientAdd = "add"
	eventStreamerClientRemove = "remove"
	eventStreamerStreaming = "streaming"
	eventStreamerClosed = "closed"
	//
	errorStreamerInvalidCommand = "invalidcmd"
	errorStreamerPoolFull = "poolfull"
	errorStreamerOffline = "offline"
)

var (
	// ErrAlreadyRunning is thrown when trying to connect a stream that is already online.
	ErrAlreadyRunning = errors.New("restreamer: service is already active")
	// ErrNotRunning is thrown trying to shut down a stopped stream.
	ErrNotRunning = errors.New("restreamer: service is not running")
	// ErrOffline is thrown when receiving a connection while the stream is offline
	ErrOffline = errors.New("restreamer: refusing connection on an offline stream")
	// ErrSlowRead is logged (not thrown) when a client can not handle the bandwidth.
	ErrSlowRead = errors.New("restreamer: send buffer overrun, increase client bandwidth")
	// ErrPoolFull is logged when the connection pool is full.
	ErrPoolFull = errors.New("restreamer: maximum number of active connections exceeded")
)

// Command is one of several possible constants.
// See StreamerCommandAdd for more information.
type Command int

const (
	// streamerCommandIgnore is a default dummy command
	streamerCommandIgnore Command = iota
	// streamerCommandStart is an internal start command, used to signal request
	// processing to commence.
	streamerCommandStart
	// StreamerCommandAdd signals a stream to add a connection.
	StreamerCommandAdd
	// StreamerCommandRemove signals a stream to remove a connection.
	StreamerCommandRemove
)

// ConnectionRequest encapsulates a request that new connection be added or removed.
type ConnectionRequest struct {
	// Command is the command to execute
	Command Command
	// Address is the remote client address
	Address string
	// Connection is the connection object
	Connection *Connection
}

// Streamer implements a TS packet multiplier,
// distributing received packets on the input queue to the output queues.
// It also handles and manages HTTP connections when added to an HTTP server.
type Streamer struct {
	// input is the input queue, accepting packets.
	// When closed, streamer is stopped and all outgoing queues along with it.
	input <-chan Packet
	// lock is the outgoing connection pool lock
	lock sync.Mutex
	// broker is a global connection broker
	broker ConnectionBroker
	// queueSize defines the maximum number of packets to queue per outgoing connection
	queueSize int
	// running reflects the state of the stream: if true, the Stream thread is running and
	// incoming connections are allowed.
	// If false, incoming connections are blocked.
	running AtomicBool
	// stats is the statistics collector for this stream
	stats Collector
	// logger is a json logger
	logger *ModuleLogger
	// request is an unbuffered queue for requests to add or remove a connection
	request chan ConnectionRequest
}

// ConnectionBroker represents a policy handler for new connections.
// It is used to determine if new connections can be accepted,
// based on arbitrary rules.
type ConnectionBroker interface {
	// Accept will be called on each incoming connection,
	// with the remote client address and the streamer that wants to accept the connection.
	Accept(remoteaddr string, streamer *Streamer) bool
	// Release will be called each time a client disconnects.
	// The streamer argument corresponds to a streamer that has previously called Accept().
	Release(streamer *Streamer)
}

// NewStreamer creates a new packet streamer.
// queue is an input packet queue.
// qsize is the length of each connection's queue (in packets).
// broker handles policy enforcement
// stats is a statistics collector object.
func NewStreamer(qsize uint, broker ConnectionBroker) (*Streamer) {
	logger := &ModuleLogger{
		Logger: &ConsoleLogger{},
		Defaults: Dict{
			"module": moduleStreamer,
		},
		AddTimestamp: true,
	}
	streamer := &Streamer{
		broker: broker,
		queueSize: int(qsize),
		running: AtomicFalse,
		stats: &DummyCollector{},
		logger: logger,
		request: make(chan ConnectionRequest),
	}
	// start the command eater
	go streamer.eatCommands()
	return streamer
}

// SetLogger assigns a logger
func (streamer *Streamer) SetLogger(logger JsonLogger) {
	streamer.logger.Logger = logger
}

// SetCollector assigns a stats collector
func (streamer *Streamer) SetCollector(stats Collector) {
	streamer.stats = stats
}

// eatCommands is started in the background to drain the command
// queue and wait for a start command, in which case it will exit.
func (streamer *Streamer) eatCommands() {
	running := true
	for running {
		select {
			case request := <-streamer.request:
				switch request.Command {
					case streamerCommandStart:
						streamer.logger.Log(Dict{
							"event": eventStreamerQueueStart,
							"message": "Stopping eater process and starting real processing",
						})
						running = false
					default:
						// Eating all other commands
				}
		}
	}
}

// Stream is the main stream multiplier loop.
// It reads data from the input queue and distributes it to the connections.
//
// This routine will block; you should run it asynchronously like this:
//
// queue := make(chan Packet, inputQueueSize)
// go func() {
//   log.Fatal(streamer.Stream(queue))
// }
//
// or simply:
//
// go streamer.Stream(queue)
func (streamer *Streamer) Stream(queue <-chan Packet) error {
	// interlock and check for availability first
	if !CompareAndSwapBool(&streamer.running, false, true) {
		return ErrAlreadyRunning
	}
	
	// create the local outgoing connection pool
	pool := make(map[*Connection]bool)
	
	// stop the eater process
	streamer.request<- ConnectionRequest{
		Command: streamerCommandStart,
	}
	
	streamer.logger.Log(Dict{
		"event": eventStreamerStart,
		"message": "Starting streaming",
	})
	
	// loop until the input channel is closed
	running := true
	for running {
		select {
			case packet, ok := <-queue:
				if ok {
					// got a packet, distribute
					//log.Printf("Got packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
					//log.Printf("Got packet (length %d)\n", len(packet))
					
					for conn, _ := range pool {
						select {
							case conn.Queue<- packet:
								// packet distributed, done
								//log.Printf("Queued packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
								
								// report the packet
								streamer.stats.PacketSent()
							default:
								// queue is full
								//log.Print(ErrSlowRead)
								
								// report the drop
								streamer.stats.PacketDropped()
						}
					}
				} else {
					// channel closed, exit
					running = false
					// and stop everything
					StoreBool(&streamer.running, false)
				}
			case request := <-streamer.request:
				switch request.Command {
					case StreamerCommandRemove:
						streamer.logger.Log(Dict{
							"event": eventStreamerClientRemove,
							"message": fmt.Sprintf("Removing client %s from pool", request.Address),
						})
						close(request.Connection.Queue)
						delete(pool, request.Connection)
					case StreamerCommandAdd:
						streamer.logger.Log(Dict{
							"event": eventStreamerClientAdd,
							"message": fmt.Sprintf("Adding client %s to pool", request.Address),
						})
						pool[request.Connection] = true
					default:
						streamer.logger.Log(Dict{
							"event": eventStreamerError,
							"error": errorStreamerInvalidCommand,
							"message": "Ignoring invalid command in started state",
						})
				}
		}
	}
	
	// clean up
	for _ = range queue {
		// drain any leftovers
	}
	for conn, _ := range pool {
		close(conn.Queue)
	}
	
	// start the command eater again
	go streamer.eatCommands()
	
	streamer.logger.Log(Dict{
		"event": eventStreamerStop,
		"message": "Ending streaming",
	})
	return nil
}

// ServeHTTP handles an incoming HTTP connection.
// Satisfies the http.Handler interface, so it can be used in an HTTP server.
func (streamer *Streamer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var conn *Connection = nil
	
	// prevent race conditions first
	if LoadBool(&streamer.running) {
		// check if the connection can be accepted
		if streamer.broker.Accept(request.RemoteAddr, streamer) {
			conn = NewConnection(writer, streamer.queueSize, request.RemoteAddr)
			conn.SetLogger(streamer.logger.Logger)
			
			streamer.request<- ConnectionRequest{
				Command: StreamerCommandAdd,
				Address: request.RemoteAddr,
				Connection: conn,
			}
		} else {
			streamer.logger.Log(Dict{
				"event": eventStreamerError,
				"error": errorStreamerPoolFull,
				"message": fmt.Sprintf("Refusing connection from %s, pool is full", request.RemoteAddr),
			})
		}
	} else {
		streamer.logger.Log(Dict{
			"event": eventStreamerError,
			"error": errorStreamerOffline,
			"message": fmt.Sprintf("Refusing connection from %s, stream is offline", request.RemoteAddr),
		})
	}
		
	if conn != nil {
		// connection will be handled, report
		streamer.stats.ConnectionAdded()
		
		streamer.logger.Log(Dict{
			"event": eventStreamerStreaming,
			"message": fmt.Sprintf("Streaming to %s", request.RemoteAddr),
			"remote": request.RemoteAddr,
		})
		conn.Serve()
		
		// done, remove the stale connection
		streamer.request<- ConnectionRequest{
			Command: StreamerCommandRemove,
			Address: request.RemoteAddr,
			Connection: conn,
		}
		// and drain the queue AFTER we have sent the shutdown signal
		for _ = range conn.Queue {
			// drain any leftovers
		}
		streamer.logger.Log(Dict{
			"event": eventStreamerClosed,
			"message": fmt.Sprintf("Connection from %s closed", request.RemoteAddr),
			"remote": request.RemoteAddr,
		})
		
		// and report
		streamer.stats.ConnectionRemoved()
		
		// also notify the broker
		streamer.broker.Release(streamer)
	} else {
		// Return a suitable error
		// TODO This should be 503 or 504, but client support seems to be poor
		// and the standards mandate nothing. Bummer.
		ServeStreamError(writer, http.StatusNotFound)
	}
}
