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
	"time"
	"net/http"
)

// Connection is a single active client connection
type Connection struct {
	// Queue is the per-connection packet queue
	Queue chan Packet
	// internal communication channel
	// for signalling connection shutdown
	shutdown chan bool
	// internal flag
	// true while the connection is up
	running bool
	// the destination socket
	writer http.ResponseWriter
}

// NewConnection creates a new connection object.
// To start sending data to a client, call Serve().
func NewConnection(destination http.ResponseWriter, qsize int) (*Connection) {
	conn := &Connection{
		Queue: make(chan Packet, qsize),
		shutdown: make(chan bool),
		running: true,
		writer: destination,
	}
	return conn
}

// Close shuts down the connection
func (conn *Connection) Close() error {
	conn.shutdown<- true
	return nil
}

// Serve starts serving data to a client,
// continuously feeding packets from the queue.
func (conn *Connection) Serve() {
	// set the content type (important)
	conn.writer.Header().Set("Content-Type", "video/mpeg")
	// a stream is always current
	conn.writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	// other headers to comply with the specs
	conn.writer.Header().Set("Accept-Range", "none")
	// use Add and Set to set more headers here
	// chunked mode should be on by default
	conn.writer.WriteHeader(http.StatusOK)
	log.Printf("Sent header\n")
	
	// see if can get notified about connection closure
	notifier, ok := conn.writer.(http.CloseNotifier)
	if !ok {
		log.Printf("Writer does not support CloseNotify")
	}
	
	// start reading packets
	for conn.running {
		select {
			case packet := <-conn.Queue:
				// packet received, log
				//log.Printf("Sending packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
				// send the packet out
				_, err := conn.writer.Write(packet)
				if err != nil {
					log.Printf("Client connection closed\n")
					conn.running = false
				}
				//log.Printf("Wrote packet of %d bytes\n", bytes)
			case <-notifier.CloseNotify():
				// connection closed while we were waiting for more data
				log.Printf("Client connection closed (while waiting)\n")
				conn.running = false
			case <-conn.shutdown:
				// and shut down
				log.Printf("Shutting down client connection\n")
				conn.running = false
		}
	}
}
