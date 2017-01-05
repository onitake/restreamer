package restreamer

import (
	"log"
	"time"
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

// a packet streamer
type Streamer struct {
	// input queue, serving packets
	input <-chan Packet
	// pool lock
	lock sync.RWMutex
	// connection pool
	connections map[*Connection]bool
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
		connections: make(map[*Connection]bool),
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
	// structural change, exclusive lock
	streamer.lock.Lock()
	for conn, _ := range streamer.connections {
		conn.Close()
	}
	streamer.lock.Unlock()
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
							log.Print(ErrSlowRead)
					}
				}
				streamer.lock.RUnlock()
			case <-time.After(1 * time.Second):
				// timeout, just cycle
			case <-streamer.shutdown:
				// and shut down
				streamer.running = false
		}
	}
	log.Printf("Ending streaming")
}

// handles an incoming connection
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
