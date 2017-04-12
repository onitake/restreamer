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

// Connection is a single active client connection.
//
// This is meant to be called directly from a ServeHTTP handler.
// No separate thread is created.
type Connection struct {
	// Queue is the per-connection packet queue
	Queue chan Packet
	// Shutdown notification channel.
	// Send true to close the connection and stop the handler.
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

// Close shuts down the streamer and all incoming connections.
// This action is asynchronous.
func (conn *Connection) Close() error {
	// signal shutdown
	conn.shutdown<- true
	return nil
}

// Serve starts serving data to a client, continuously feeding packets from the queue.
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
	log.Printf("Sent header")
	
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
					log.Printf("Client connection closed")
					conn.running = false
				}
				//log.Printf("Wrote packet of %d bytes\n", bytes)
			case <-notifier.CloseNotify():
				// connection closed while we were waiting for more data
				log.Printf("Client connection closed (while waiting)")
				conn.running = false
			case <-conn.shutdown:
				// and shut down
				log.Printf("Shutting down client connection")
				conn.running = false
		}
	}
	
	// drain the shutdown channel
	select {
		case <-streamer.shutdown:
		default:
	}
}

// ServeStreamError returns an appropriate error response to the client.
func ServeStreamError(writer http.ResponseWriter, status int) {
	// set the content type (important)
	writer.Header().Set("Content-Type", "video/mpeg")
	// a stream is always current
	writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	// other headers to comply with the specs
	writer.Header().Set("Accept-Range", "none")
	// ...and the application-supplied status code
	writer.WriteHeader(http.StatusNotFound)
}
