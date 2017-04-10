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
	"log"
	//"time"
	"sync"
	"errors"
	"net/http"
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

// Streamer implements a TS packet multiplier,
// distributing received packets on the input queue to the output queues.
// It also handles and manages HTTP connections when added to an HTTP server.
type Streamer struct {
	// input is the input queue, accepting packets
	input <-chan Packet
	// lock is the outgoing connection pool lock
	lock sync.RWMutex
	// connections is the outgoing connection pool
	connections map[*Connection]bool
	// shutdown is the shutdown notification channel.
	// Send true to close all downstream connections and stop the multiplier.
	shutdown chan bool
	// running is the global running state flag;
	// setting it to false causes a shutdown on the
	// the next queue run - but you should not do this.
	// Better just use the shutdown channel.
	running bool
	// broker is a global connection broker
	broker ConnectionBroker
	// queueSize defines the maximum number of packets to queue per per connection
	queueSize int
	// stats is the statistics collector for this stream
	stats Collector
	// logger is a json logger
	logger JsonLogger
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
func NewStreamer(queue <-chan Packet, qsize uint, broker ConnectionBroker) (*Streamer) {
	streamer := &Streamer{
		input: queue,
		connections: make(map[*Connection]bool),
		shutdown: make(chan bool),
		running: false,
		broker: broker,
		queueSize: int(qsize),
		stats: &DummyCollector{},
		logger: &DummyLogger{},
	}
	return streamer
}

// SetLogger assigns a logger
func (streamer *Streamer) SetLogger(logger JsonLogger) {
	streamer.logger = logger
}

// SetCollector assigns a stats collector
func (streamer *Streamer) SetCollector(stats Collector) {
	streamer.stats = stats
}

// Close shuts down the streamer and all incoming connections.
// The stream is restartable, you may call Connect() again later.
//
// Most effects of the shutdown are immediate, but it may take a moment until
// all downstream connections are closed.
// Calling Connect() right after Close() returns should be safe, however.
func (streamer *Streamer) Close() error {
	if (streamer.running) {
		// prevent further connections
		streamer.running = false
		// signal shutdown
		streamer.shutdown<- true
		
		// acquire the exclusive lock
		streamer.lock.Lock()
		// clean up
		for conn, _ := range streamer.connections {
			conn.Close()
		}
		// reset the connection pool
		streamer.connections = make(map[*Connection]bool)
		streamer.lock.Unlock()
		
		return nil
	}
	return ErrNotRunning
}

// Connect starts serving and streaming.
func (streamer *Streamer) Connect() error {
	if (!streamer.running) {
		streamer.running = true
		go streamer.stream()
		return nil
	}
	return ErrAlreadyRunning
}

// stream is the internal stream multiplier loop.
// It reads data from the input and distributes it to the connections.
// Will be called asynchronously from Connect()
func (streamer *Streamer) stream() {
	log.Printf("Starting streaming")
	// local run flag to avoid race conditions
	running := true
	for running && streamer.running {
		select {
			case packet := <-streamer.input:
				// got a packet, distribute
				//log.Printf("Got packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				
				// content distribution only, lock in non-exclusive read mode
				streamer.lock.RLock()
				for conn, _ := range streamer.connections {
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
				streamer.lock.RUnlock()
			case <-streamer.shutdown:
				// stop no matter what the global state is
				running = false
		}
	}
	log.Printf("Ending streaming")
}

// ServeHTTP handles an incoming HTTP connection.
// Satisfies the http.Handler interface, so it can be used in an HTTP server.
func (streamer *Streamer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var conn *Connection = nil
	
	// prevent race conditions first
	if streamer.running {
		// check if the connection can be accepted
		if streamer.broker.Accept(request.RemoteAddr, streamer) {
			// structural change, exclusive lock
			streamer.lock.Lock()
			// instantiate
			conn = NewConnection(writer, streamer.queueSize)
			streamer.connections[conn] = true
			// and release
			streamer.lock.Unlock()
		} else {
			log.Printf("Refusing connection from %s, pool is full", request.RemoteAddr)
		}
	} else {
		log.Printf("Refusing connection from %s, stream is offline", request.RemoteAddr)
	}
		
	if conn != nil {
		// connection will be handled, report
		streamer.stats.ConnectionAdded()
		
		log.Printf("Streaming to %s\n", request.RemoteAddr);
		conn.Serve()
		
		// done, remove the stale connection
		streamer.lock.Lock()
		delete(streamer.connections, conn)
		streamer.lock.Unlock()
		log.Printf("Connection from %s closed\n", request.RemoteAddr);
		
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
