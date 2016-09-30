
type Server struct {
	
}

func NewStream(pullurl string, packets int) (*Stream) {
	server := &http.Server {
		Addr: cfg.Listen,
		Handler: mux,
		ReadTimeout: time.Duration(cfg.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
		//MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(server.ListenAndServe())
}

// connects the data source
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
