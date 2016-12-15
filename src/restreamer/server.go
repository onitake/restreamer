package restreamer

import (
	"log"
	"time"
	"errors"
	"net"
	"net/http"
)

var (
	// ErrAlreadyRunning is thrown when trying to listen on
	// an active server twice.
	ErrAlreadyRunning = errors.New("restreamer: service is already active")
	// ErrSlowRead is logged (not thrown) when a client can not
	// handle the bandwidth
	ErrSlowRead = errors.New("restreamer: send buffer overrun, increase client bandwidth")
)

// a packet server
type StreamingServer struct {
	// we are an HTTP server
	*http.Server
	// expose the listener to allow closing it
	Listener net.Listener
	// input queue, serving packets
	Input <-chan Packet
	// list of connections
	Connections []*Connection
	// internal communication channel
	// for signalling global shutdown
	Shutdown chan bool
	// global running flag
	Running bool
	// limit on the number of connections
	MaxConnections int
	// the maximum number of packets to queue per per connection
	QueueSize int
}

// create a new packet server
// listenon: the listen address:port tuple (use :http to listen on all interfaces)
// queue: an input packet queue
// timeout: the timeout for write operations
// maxconn: the maximum number of connections to accept concurrently
// qsize: the length of each connection's queue (in packets)
func NewServer(listenon string, queue <-chan Packet, timeout int, maxconn int, qsize int) (*StreamingServer) {
	if listenon == "" {
		listenon = ":http"
	}
	server := &StreamingServer{
		Server: &http.Server{
			Addr: listenon,
			ReadTimeout: time.Duration(timeout) * time.Second,
			WriteTimeout: time.Duration(timeout) * time.Second,
			//MaxHeaderBytes: 1 << 20,
		},
		Listener: nil,
		Input: queue,
		Connections: make([]*Connection, maxconn, 0),
		Shutdown: make(chan bool),
		Running: false,
		MaxConnections: maxconn,
		QueueSize: qsize,
	}
	server.Server.Handler = server
	return server
}

// shuts down the server
// and all incoming connections
func (server *StreamingServer) Close() {
	server.Running = false
	server.Shutdown<- true
	for _, conn := range server.Connections {
		conn.Shutdown<- true
	}
	server.Listener.Close()
}

// starts serving and streaming
func (server *StreamingServer) ListenAndServe() error {
	if (!server.Running) {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			return err
		}
		server.Listener = listener
		
		server.Running = true
		go server.Stream()
		
		// perhaps we should wrap a tcpKeepAliveListener here?
		return server.Serve(server.Listener)
	}
	return ErrAlreadyRunning
}

// stream multiplier
// reads data from the input and distributes it to the connections
func (server *StreamingServer) Stream() {
	for server.Running {
		select {
			case packet := <-server.Input:
				// got a packet, distribute
				for _, conn := range server.Connections {
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
			case <-server.Shutdown:
				// and shut down
				server.Running = false
		}
	}
}

// handles and defers an incoming connection
func (server *StreamingServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if (len(server.Connections) < server.MaxConnections) {
		conn := NewConnection(server.QueueSize)
		server.Connections = append(server.Connections, conn)
	} else {
		log.Printf("Error: maximum number of connections reached")
	}
}
