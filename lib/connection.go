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
	"time"
	"net/http"
)

const (
	moduleConnection = "connection"
	//
	eventConnectionDebug = "debug"
	eventConnectionError = "error"
	eventHeaderSent = "headersent"
	eventConnectionClosed = "closed"
	eventConnectionClosedWait = "closedwait"
	eventConnectionShutdown = "shutdown"
	eventConnectionDone = "done"
	//
	errorConnectionNotFlushable = "noflush"
	errorConnectionNoCloseNotify = "noclosenotify"
)

// Connection is a single active client connection.
//
// This is meant to be called directly from a ServeHTTP handler.
// No separate thread is created.
type Connection struct {
	// Queue is the per-connection packet queue
	Queue chan Packet
	// the destination socket
	writer http.ResponseWriter
	// needed for flushing
	flusher http.Flusher
	// logger is a json logger
	logger *ModuleLogger
}

// NewConnection creates a new connection object.
// To start sending data to a client, call Serve().
//
// clientaddr should point to the remote address of the connecting client
// and will be used for logging.
func NewConnection(destination http.ResponseWriter, qsize int, clientaddr string) (*Connection) {
	logger := &ModuleLogger{
		Logger: &ConsoleLogger{},
		Defaults: Dict{
			"module": moduleConnection,
			"remote": clientaddr,
		},
		AddTimestamp: true,
	}
	flusher, ok := destination.(http.Flusher)
	if !ok {
		logger.Log(Dict{
			"event": eventConnectionError,
			"error": errorConnectionNotFlushable,
			"message": "ResponseWriter is not flushable!",
		})
	}
	conn := &Connection{
		Queue: make(chan Packet, qsize),
		writer: destination,
		flusher: flusher,
		logger: logger,
	}
	return conn
}

// SetLogger assigns a logger
func (conn *Connection) SetLogger(logger JsonLogger) {
	conn.logger.Logger = logger
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
	if conn.flusher != nil {
		conn.flusher.Flush()
	}
	conn.logger.Log(Dict{
		"event": eventHeaderSent,
		"message": "Sent header",
	})
	
	// see if can get notified about connection closure
	notifier, ok := conn.writer.(http.CloseNotifier)
	if !ok {
		conn.logger.Log(Dict{
			"event": eventConnectionError,
			"error": errorConnectionNoCloseNotify,
			"message": "Writer does not support CloseNotify",
		})
	}
	
	// start reading packets
	running := true
	for running {
		select {
			case packet, ok := <-conn.Queue:
				if ok {
					// packet received, log
					//log.Printf("Sending packet (length %d):\n%s\n", len(packet), hex.Dump(packet))
					// send the packet out
					_, err := conn.writer.Write(packet)
					if err == nil {
						if conn.flusher != nil {
							conn.flusher.Flush()
						}
					} else {
						conn.logger.Log(Dict{
							"event": eventConnectionClosed,
							"message": "Downstream connection closed",
						})
						running = false
					}
					//log.Printf("Wrote packet of %d bytes\n", bytes)
				} else {
					// channel closed, exit
					conn.logger.Log(Dict{
						"event": eventConnectionShutdown,
						"message": "Shutting down client connection",
					})
					running = false
				}
			case <-notifier.CloseNotify():
				// connection closed while we were waiting for more data
				conn.logger.Log(Dict{
					"event": eventConnectionClosedWait,
					"message": "Downstream connection closed (while waiting)",
				})
				running = false
		}
	}
	
	// we cannot drain the channel here, as it might not be closed yet.
	// better let our caller handle closure and draining.
	
	conn.logger.Log(Dict{
		"event": eventConnectionDone,
		"message": "Streaming finished",
	})
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
