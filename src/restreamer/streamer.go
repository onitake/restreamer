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
	// ErrAlreadyrunning is thrown when trying to listen on
	// an active streamer twice.
	ErrAlreadyrunning = errors.New("restreamer: service is already active")
	// ErrSlowRead is logged (not thrown) when a client can not
	// handle the bandwidth
	ErrSlowRead = errors.New("restreamer: send buffer overrun, increase client bandwidth")
	// ErrPoolFull is logged when the connection pool is full.
	ErrPoolFull = errors.New("restreamer: maximum number of active connections exceeded")
)

// Streamer implements a TS packet multiplier,
// distributing received packets on the input queue to the output queues.
// It also handles and manages HTTP connections when added to an HTTP server.
type Streamer struct {
	// input is the input queue, serving packets
	input <-chan Packet
	// lock is the outgoing queue pool lock
	lock sync.RWMutex
	// connections is the outgoing connection pool
	connections map[*Connection]bool
	// shutdown is an internal communication channel
	// for signalling global shutdown
	shutdown chan bool
	// running is the global running state flag;
	// setting it to false causes a shutdown on the
	// the next queue run - but you should not do this.
	// Better just send something to the shutdown channel.
	running bool
	// maxconnections is the limit on the number of connections
	maxconnections int
	// queueSize defines the maximum number of packets to queue per per connection
	queueSize int
}

// NewStreamer creates a new packet streamer.
// listenon: the listen address:port tuple - use ":http" or "" to listen on port 80 on all interfaces
// queue: an input packet queue
// maxconn: the maximum number of connections to accept concurrently
// qsize: the length of each connection's queue (in packets)
func NewStreamer(queue <-chan Packet, maxconn uint, qsize uint) (*Streamer) {
	streamer := &Streamer{
		input: queue,
		connections: make(map[*Connection]bool),
		shutdown: make(chan bool),
		running: false,
		maxconnections: int(maxconn),
		queueSize: int(qsize),
	}
	return streamer
}

// Close shuts the streamer and all incoming connections down.
func (streamer *Streamer) Close() error {
	streamer.shutdown<- true
	// structural change, exclusive lock
	streamer.lock.Lock()
	for conn, _ := range streamer.connections {
		conn.Close()
	}
	streamer.lock.Unlock()
	return nil
}

// Connect starts serving and streaming.
func (streamer *Streamer) Connect() error {
	if (!streamer.running) {
		streamer.running = true
		go streamer.stream()
		return nil
	}
	return ErrAlreadyrunning
}

// stream is the internal stream multiplier loop.
// It reads data from the input and distributes it to the connections.
// Will be called asynchronously from Connect()
func (streamer *Streamer) stream() {
	log.Printf("Starting streaming")
	for streamer.running {
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
						default:
							// queue is full
							//log.Print(ErrSlowRead)
					}
				}
				streamer.lock.RUnlock()
			case <-streamer.shutdown:
				// and shut down
				streamer.running = false
		}
	}
	log.Printf("Ending streaming")
}

// ServeHTTP handles an incoming connection.
// Satisfies the http.Handler interface, so it can be used in an HTTP server.
func (streamer *Streamer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var conn *Connection
	
	// structural change, exclusive lock
	streamer.lock.Lock()
	log.Printf("Got a connection from %s, number of active connections is %d, max number of connections is %d\n", request.RemoteAddr, len(streamer.connections), streamer.maxconnections)
	// check if we still have free connections first
	if (len(streamer.connections) < streamer.maxconnections) {
		conn = NewConnection(writer, streamer.queueSize)
		streamer.connections[conn] = true
	} else {
		// no error return, so just log
		log.Print(ErrPoolFull)
	}
	streamer.lock.Unlock()
	
	if conn != nil {
		log.Printf("Streaming to %s\n", request.RemoteAddr);
		conn.Serve()
		
		// done, remove the stale connection
		streamer.lock.Lock()
		delete(streamer.connections, conn)
		streamer.lock.Unlock()
		log.Printf("Connection from %s closed\n", request.RemoteAddr);
	}
}
