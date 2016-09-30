package main

import (
	"os"
	"log"
	"fmt"
	"time"
	"sync"
	"net/http"
	"encoding/json"
)
//	"encoding/hex"

const (
	// TS packet size
	PACKET_SIZE = 188
	// TS packet size with padding
	BUFFER_SIZE = PACKET_SIZE + 4
	// TS packet synchronization byte
	SYNC_BYTE = 0x47
)

// one TS packet
// 188 bytes long
// starts with 0x47
// 4 bytes of padding
// yes, this is a type alias to a byte slice
// use NewPacket to construct a packet, optionally
// by copying data from a data buffer or another packet
type Packet []byte

// creates a new packet
// and optionally fills it with data
func NewPacket(data []byte) Packet {
	// allocate a padded data buffer and create
	// a slice of the correct size from it
	p := make(Packet, PACKET_SIZE, BUFFER_SIZE)
	if data != nil {
		copy(p, data)
	}
	return p
}

// generic HTTP client error
type HttpClientError struct {
	Err string
}

// returns the error message
func (e HttpClientError) Error() string {
	return e.Err
}

// a streaming HTTP client
type StreamingHttpClient struct {
	// the URL to GET
	Url string
	// the response, including the body reader
	Socket *http.Response
}

// construct a new streaming HTTP client
// call Connect() to connect the socket, send the request
// and read the response headers
func NewHttpClient(url string) *StreamingHttpClient {
	c := &StreamingHttpClient {
		Url: url,
	}
	return c
}

// connects the socket and sends the HTTP request
func (c *StreamingHttpClient) Connect() error {
	if c.Socket != nil {
		c.Close()
	}
	client := &http.Client {
		Timeout: 10 * time.Second,
	}
	r, err := client.Get(c.Url)
	if err != nil {
		return err
	}
	c.Socket = r
	return nil
}

// reads some body data from the connection
func (c *StreamingHttpClient) Read(p []byte) (n int, err error) {
	if c.Socket == nil {
		return 0, HttpClientError { "Socket not connected" }
	}
	return c.Socket.Body.Read(p)
}

// closes the connection
func (c *StreamingHttpClient) Close() error {
	if c.Socket != nil {
		err := c.Socket.Body.Close()
		c.Socket = nil
		return err
	}
	return nil
}

// returns the HTTP status code or 0 if not connected
func (c *StreamingHttpClient) StatusCode() int {
	if c.Socket != nil {
		return c.Socket.StatusCode
	}
	return 0
}

// returns the HTTP status message or the empty string if not connected
func (c *StreamingHttpClient) Status() string {
	if c.Socket != nil {
		return c.Socket.Status
	}
	return ""
}

// returns true if the socket is still connected
func (c *StreamingHttpClient) Connected() bool {
	return c.Socket != nil
}

// a source stream
// contains a ring buffer for packets,
// the head and tail points in the buffer
// the size and allocation length of the buffer,
// the waterline to use as ahead/behind indicator
// and the HTTP stream source
type Stream struct {
	// the ring buffer
	RingBuffer []Packet
	// the insertion point
	Tail int
	// the fill level
	Size int
	// the median line that consumers should follow
	Tideline int
	// the data source
	Source *StreamingHttpClient
	// the ring buffer semaphore
	DataAvailable *sync.Cond
}

// creates a new HTTP get stream from a pull URL and
// a packet queue size
// the stream is connected and ready to use
func NewStream(pullurl string, packets int) (*Stream) {
	return &Stream {
		RingBuffer: make([]Packet, packets),
		Tail: 0,
		Size: packets,
		Tideline: packets / 2,
		Source: NewHttpClient(pullurl),
		DataAvailable: sync.NewCond(&sync.Mutex { }),
	}
}

// connects the data source and spawns the reader
func (s *Stream) Connect() error {
	log.Printf("Connecting to %s\n", s.Source.Url)

	err := s.Source.Connect()
	if err != nil {
		return err
	}
	if s.Source.StatusCode() != http.StatusOK && s.Source.StatusCode() != http.StatusPartialContent {
		return HttpClientError { fmt.Sprintf("Unsupported response code: %s", s.Source.Status()) }
	}
	
	// start streaming
	go s.Pull()
	
	return nil
}

// inserts a packet at the tail
// and updates tail, size and tide line
// overwrites the oldest packet
func (s *Stream) Put(p Packet) {
	t := s.Tail
	//log.Printf("Inserting packet at %d\n", t)
	s.RingBuffer[t] = p
	t++
	if t >= len(s.RingBuffer) {
		t = 0
	}
	s.Tail = t
	if s.Size < len(s.RingBuffer) {
		s.Size++
		s.Tideline = s.Size / 2
	} else {
		w := s.Tideline
		w++
		if w >= len(s.RingBuffer) {
			w = 0
		}
		s.Tideline = w
	}
	// signal all waiters
	s.DataAvailable.Broadcast()
	//log.Printf("Tail is now %d, size is %d and tide line is %d\n", s.Tail, s.Size, s.Tideline)
}

// returns a packet and the next extraction point
// from the ring buffer
// the extraction point must be specified
// blocks or returns nil if the point is too close to the tail
func (s *Stream) Get(n int, block bool) (Packet, int) {
	l := s.Size
	//log.Printf("Extracting packet at %d\n", n)
	if n > l {
		//log.Printf("Extraction point out of bounds: %d (limit %d)\n", n, l)
		return nil, n
	}
	t := s.Tail
	// TODO find a better heuristic for overruns
	if n == t {
		if block {
			//log.Printf("Overrun trying to read from the queue, blocking\n")
			s.DataAvailable.L.Lock()
			s.DataAvailable.Wait()
			s.DataAvailable.L.Unlock()
		} else {
			log.Printf("Overrun trying to read from the queue, return nil\n")
			return nil, n
		}
	}
	p := s.RingBuffer[n]
	n++
	if n >= len(s.RingBuffer) {
		n = 0
	}
	return p, n
}

// reads packets in a loop
// and adds them to the ring buffer
func (s *Stream) Pull() {
	log.Printf("Reading stream from %s\n", s.Source.Url)
	
	for {
		p, err := s.ReadPacket()
		if err != nil {
			//log.Print(err)
			//break
		}
		//log.Printf("Got a packet (length %d):\n%s\n", len(p), hex.Dump(p))
		s.Put(p)
	}
	
	log.Printf("Socket for stream %s closed, exiting\n", s.Source.Url)
}

// reads a single packet and returns it
func (s *Stream) ReadPacket() (Packet, error) {
	// read 188 bytes ahead (assume we are at the start of a packet)
	p := NewPacket(nil)
	_, err := s.Source.Read(p[:PACKET_SIZE])
	if err != nil {
		return nil, err
	}
	//log.Printf("Read %d bytes\n", n)
	
	// quick check for the sync byte 0x47
	if p[0] != SYNC_BYTE {
		// nope, scan
		sync := -1
		for i, b := range p[:PACKET_SIZE] {
			if b == SYNC_BYTE {
				// found, very good
				sync = i
				break
			}
		}
		// return an error if not found
		if sync == -1 {
			return nil, HttpClientError { "Can't find sync byte, check your stream" }
		}
		// if the sync byte was not at the beginning,
		// resize the slice and append the remaining data
		// this should happen only when the stream is out of sync,
		// so performance impact is minimal
		p = NewPacket(p[sync:])
		offset := PACKET_SIZE - sync
		_, err := s.Source.Read(p[offset:PACKET_SIZE])
		if err != nil {
			return nil, err
		}
		//log.Printf("Appended %d bytes\n", n)
	}
	
	// and done
	return p, nil
}

// handles incoming connections
// continuously streams packets from the ring buffer,
// trying to keep close to the tide line
func (s *Stream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving incoming connection from %s\n", r.RemoteAddr);
	
	// get the starting point
	n := s.Tideline
	
	// set the content type (important)
	w.Header().Set("Content-Type", "video/mpeg")
	// use Add and Set to set more headers here
	w.WriteHeader(http.StatusOK)
	
	// read packets
	for {
		// read a packet from the ring buffer, blocking until data is available
		var p Packet
		for p == nil {
			// bail out if the connection is closed
			if !s.Source.Connected() {
				log.Fatal("Incoming connection is closed\n")
				return
			}
			p, n = s.Get(n, true)
		}
		
		// log the packet
		//log.Printf("Got a packet (length %d):\n%s\n", len(p), hex.Dump(p))
		
		// send the packet out
		_, err := w.Write(p[:PACKET_SIZE])
		if err != nil {
			log.Printf("Connection from %s closed\n", r.RemoteAddr)
			return
		}
		//log.Printf("Wrote packet of %d bytes\n", b)
	}
}

type Configuration struct {
	Listen string
	Timeout int
	Buffer int
	Streams []struct {
		Serve string
		Remote string
	}
}

func main() {
	var cfgname string
	if len(os.Args) > 1 {
		cfgname = os.Args[1]
	} else {
		cfgname = "server.json"
	}
	
	cfgfile, err := os.Open(cfgname)
	if err != nil {
		log.Fatal("Can't read configuration from server.json\n", err)
	}
	decoder := json.NewDecoder(cfgfile)
	cfg := Configuration { }
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal("Error parsing configuration\n", err)
	}
	cfgfile.Close()

	log.Printf("Listen = %s\n", cfg.Listen)
	log.Printf("Timeout = %d\n", cfg.Timeout)

	i := 0
	mux := http.NewServeMux()
	for _, m := range cfg.Streams {
		log.Printf("Testing stream %s\n", m.Remote)
		s := NewStream(m.Remote, cfg.Buffer)
		err := s.Connect()
		if err == nil {
			mux.Handle(m.Serve, s)
			log.Printf("Handled connection %d", i)
			i++
		} else {
			log.Print(err)
		}
	}
	
	if i == 0 {
		log.Fatal(HttpClientError { "No streams available" })
	} else {
		log.Print("Starting server")
		server := &http.Server {
			Addr: cfg.Listen,
			Handler: mux,
			ReadTimeout: time.Duration(cfg.Timeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
			//MaxHeaderBytes: 1 << 20,
		}
		log.Fatal(server.ListenAndServe())
	}
}
