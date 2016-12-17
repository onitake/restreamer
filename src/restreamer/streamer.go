package restreamer

import (
	"log"
	"time"
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
)

// a packet streamer
type Streamer struct {
	// input queue, serving packets
	input <-chan Packet
	// list of connections
	connections []*Connection
	// internal communication channel
	// for signalling global shutdown
	shutdown chan bool
	// global running flag
	running bool
	// limit on the number of connections
	maxconnections int
	// the maximum number of packets to queue per per connection
	queueSize int
}

// create a new packet streamer
// listenon: the listen address:port tuple - use ":http" or "" to listen on port 80 on all interfaces
// queue: an input packet queue
// maxconn: the maximum number of connections to accept concurrently
// qsize: the length of each connection's queue (in packets)
func NewStreamer(queue <-chan Packet, maxconn int, qsize int) (*Streamer) {
	streamer := &Streamer{
		input: queue,
		connections: make([]*Connection, 0, maxconn),
		shutdown: make(chan bool),
		running: false,
		maxconnections: maxconn,
		queueSize: qsize,
	}
	return streamer
}

// shuts down the streamer
// and all incoming connections
func (streamer *Streamer) Close() error {
	streamer.shutdown<- true
	for _, conn := range streamer.connections {
		conn.Close()
	}
	return nil
}

// starts serving and streaming
func (streamer *Streamer) Connect() error {
	if (!streamer.running) {
		streamer.running = true
		go streamer.stream()
		return nil
	}
	return ErrAlreadyrunning
}

// stream multiplier
// reads data from the input and distributes it to the connections
func (streamer *Streamer) stream() {
	log.Printf("Starting to stream")
	for streamer.running {
		select {
			case packet := <-streamer.input:
				// got a packet, distribute
				for _, conn := range streamer.connections {
					select {
						case conn.Queue<- packet:
							// distributed packet, done
						default:
							// queue is full
							log.Println(ErrSlowRead)
					}
				}
			case <-time.After(1 * time.Second):
				// timeout, just cycle
			case <-streamer.shutdown:
				// and shut down
				streamer.running = false
		}
	}
	log.Printf("Ending streaming")
}

// handles and defers an incoming connection
func (streamer *Streamer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	log.Printf("Got a connection from %s, number of active connections is %d, max number of connections is %d\n", request.RemoteAddr, len(streamer.connections), streamer.maxconnections)
	if (len(streamer.connections) < streamer.maxconnections) {
		conn := NewConnection(streamer.queueSize)
		streamer.connections = append(streamer.connections, conn)
		conn.ServeHTTP(writer, request)
	} else {
		log.Printf("Error: maximum number of connections reached")
	}
}
