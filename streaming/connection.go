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
	"github.com/onitake/restreamer/mpegts"
	"github.com/onitake/restreamer/util"
	"net/http"
	"time"
)

const (
	moduleConnection = "connection"
	//
	eventConnectionDebug      = "debug"
	eventConnectionError      = "error"
	eventHeaderSent           = "headersent"
	eventConnectionClosed     = "closed"
	eventConnectionClosedWait = "closedwait"
	eventConnectionShutdown   = "shutdown"
	eventConnectionDone       = "done"
	//
	errorConnectionNotFlushable  = "noflush"
	errorConnectionNoCloseNotify = "noclosenotify"
)

// Connection is a single active client connection.
//
// This is meant to be called directly from a ServeHTTP handler.
// No separate thread is created.
type Connection struct {
	// Queue is the per-connection packet queue
	Queue chan mpegts.Packet
	// the destination socket
	writer http.ResponseWriter
	// logger is a json logger
	logger *util.ModuleLogger
}

// NewConnection creates a new connection object.
// To start sending data to a client, call Serve().
//
// clientaddr should point to the remote address of the connecting client
// and will be used for logging.
func NewConnection(destination http.ResponseWriter, qsize int, clientaddr string) *Connection {
	logger := &util.ModuleLogger{
		Logger: &util.ConsoleLogger{},
		Defaults: util.Dict{
			"module": moduleConnection,
			"remote": clientaddr,
		},
		AddTimestamp: true,
	}
	conn := &Connection{
		Queue:   make(chan mpegts.Packet, qsize),
		writer:  destination,
		logger:  logger,
	}
	return conn
}

// SetLogger assigns a logger
func (conn *Connection) SetLogger(logger util.JsonLogger) {
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
	// suppress caching by intermediate proxies
	conn.writer.Header().Set("Cache-Control", "no-cache,no-store,no-transform")
	// use Add and Set to set more headers here
	// chunked mode should be on by default
	conn.writer.WriteHeader(http.StatusOK)
	// try to flush the header
	flusher, ok := conn.writer.(http.Flusher)
	if !ok {
		conn.logger.Log(util.Dict{
			"event":   eventConnectionError,
			"error":   errorConnectionNotFlushable,
			"message": "ResponseWriter is not flushable!",
		})
	} else {
		flusher.Flush()
	}
	conn.logger.Log(util.Dict{
		"event":   eventHeaderSent,
		"message": "Sent header",
	})

	// see if can get notified about connection closure
	notifier, ok := conn.writer.(http.CloseNotifier)
	if !ok {
		conn.logger.Log(util.Dict{
			"event":   eventConnectionError,
			"error":   errorConnectionNoCloseNotify,
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
				// NOTE we shouldn't flush here, to avoid swamping the kernel with syscalls.
				// see https://golang.org/pkg/net/http/?m=all#response.Write for details
				// on how Go buffers HTTP responses (hint: a 2KiB bufio and a 4KiB bufio)
				if err != nil {
					conn.logger.Log(util.Dict{
						"event":   eventConnectionClosed,
						"message": "Downstream connection closed",
					})
					running = false
				}
				//log.Printf("Wrote packet of %d bytes\n", bytes)
			} else {
				// channel closed, exit
				conn.logger.Log(util.Dict{
					"event":   eventConnectionShutdown,
					"message": "Shutting down client connection",
				})
				running = false
			}
		case <-notifier.CloseNotify():
			// connection closed while we were waiting for more data
			conn.logger.Log(util.Dict{
				"event":   eventConnectionClosedWait,
				"message": "Downstream connection closed (while waiting)",
			})
			running = false
		}
	}

	// we cannot drain the channel here, as it might not be closed yet.
	// better let our caller handle closure and draining.

	conn.logger.Log(util.Dict{
		"event":   eventConnectionDone,
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
	// suppress caching by intermediate proxies
	writer.Header().Set("Cache-Control", "no-cache,no-store,no-transform")
	// ...and the application-supplied status code
	writer.WriteHeader(http.StatusNotFound)
}
