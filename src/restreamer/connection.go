package restreamer

import (
	"log"
	"time"
	"net/http"
)

// a single active connection
type Connection struct {
	// per-connection packet queue
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

// creates a new connection object.
// note that HTTP is not handled here, only data is transmitted.
// call Serve to start streaming
func NewConnection(destination http.ResponseWriter, qsize int) (*Connection) {
	conn := &Connection{
		Queue: make(chan Packet, qsize),
		shutdown: make(chan bool),
		running: true,
		writer: destination,
	}
	return conn
}

// closes a connection
func (conn *Connection) Close() error {
	conn.shutdown<- true
	return nil
}

// serves a connection,
// continuously streaming packets from the queue
func (conn *Connection) Serve() {
	// set the content type (important)
	conn.writer.Header().Set("Content-Type", "video/mpeg")
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
			case <-time.After(1 * time.Second):
				// timeout, just cycle
			case <-conn.shutdown:
				// and shut down
				log.Printf("Shutting down client connection\n")
				conn.running = false
		}
	}
}
