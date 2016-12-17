package restreamer

import (
	"time"
	"net"
	"net/http"
)

type StreamingServer struct {
	// yes, we are a server
	http.Server
	// store the listener to allow closing it
	listener net.Listener
}

func NewServer(listenon string, timeout int) (*StreamingServer) {
	if listenon == "" {
		listenon = ":http"
	}
	server := &StreamingServer{
		Server: http.Server{
			Addr: listenon,
			ReadTimeout: time.Duration(timeout) * time.Second,
			WriteTimeout: time.Duration(timeout) * time.Second,
			//MaxHeaderBytes: 1 << 20,
		},
		listener: nil,
	}
	//server.Server.Handler = server
	return server
}

func (server *StreamingServer) ListenAndServe() error { 
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	return server.Serve(listener)
}

func (server *StreamingServer) Serve(listener net.Listener) error {
	if (server.listener != nil) {
		server.listener = listener
		
		// perhaps we should wrap a tcpKeepAliveListener here?
		return server.Serve(server.listener)
	}
	return nil
}
