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

package main

import (
	"os"
	"log"
	"sync"
	"net/http"
	"encoding/json"
	"restreamer"
)

// Configuration is a representation of the configurable settings.
// These are normally read from a JSON file and deserialized by
// the builtin marshaler.
type Configuration struct {
	// the interface to listen on
	Listen string
	// the connection timeout
	// (both input and output)
	Timeout uint
	// the reconnect delay
	Reconnect uint
	// the maximum number of packets
	// on the input buffer
	InputBuffer uint
	// the size of the output buffer
	// per connection
	// note that each connection will
	// eat at least OutputBuffer * 192 bytes
	// when the queue is full, so
	// you should adjust the value according
	// to the amount of RAM available
	OutputBuffer uint
	// the maximum total number of concurrent connections
	MaxConnections uint
	// set to true to disable statistics
	NoStats bool
	// the list of streams
	Resources []struct {
		// the resource type
		Type string
		// the API type
		Api string
		// the local URL to serve this stream under
		Serve string
		// the upstream URL or API argument
		Remote string
		// the cache time in seconds
		Cache uint
	}
}

// accessController implements a connection broker that limits
// the maximum number of concurrent connections.
type accessController struct {
	// maxconnections is a global limit on the number of connections.
	maxconnections uint
	// lock to protect the connection counter
	lock sync.Mutex
	// connections contains the number of active connections.
	// must be accessed atomically.
	connections uint
}

// newAccessController creates a connection broker object that
// handles access control according to the number of connected clients.
func newAccessController(maxconnections uint) *accessController {
	return &accessController{
		maxconnections: maxconnections,
	}
}

// Accept is the access control handler.
func (control *accessController) Accept(remoteaddr string, id interface{}) bool {
	accept := false
	// protect concurrent access
	control.lock.Lock()
	if control.connections < control.maxconnections {
		// and increase the counter
		control.connections++
		accept = true
	}
	control.lock.Unlock()
	// print some info
	if accept {
		log.Printf("Accepted connection from %s @%p, active=%d, max=%d\n", remoteaddr, id, control.connections, control.maxconnections)
	} else {
		log.Printf("Denied connection from %s @%p, active=%d, max=%d\n", remoteaddr, id, control.connections, control.maxconnections)
	}
	// return the result
	return accept
}

func (control *accessController) Release(id interface{}) {
	remove := false
	// protect concurrent access
	control.lock.Lock()
	if control.connections > 0 {
		// and decrease the counter
		control.connections--
		remove = true
	}
	control.lock.Unlock()
	if remove {
		log.Printf("Removed connection @%p\n", id)
	} else {
		log.Printf("Error, no connection to remove @%p\n", id)
	}
}

func main() {
	var configname string
	if len(os.Args) > 1 {
		configname = os.Args[1]
	} else {
		configname = "restreamer.json"
	}
	
	configfile, err := os.Open(configname)
	if err != nil {
		log.Fatal("Can't read configuration from server.json: ", err)
	}
	decoder := json.NewDecoder(configfile)
	config := Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Error parsing configuration: ", err)
	}
	configfile.Close()

	log.Printf("Listen = %s", config.Listen)
	log.Printf("Timeout = %d", config.Timeout)
	
	var stats restreamer.Statistics
	if config.NoStats {
		stats = restreamer.NewDummyStatistics()
	} else {
		stats = restreamer.NewStatistics(config.MaxConnections)
	}
	
	controller := newAccessController(config.MaxConnections)
	
	clients := make(map[string]*restreamer.Client)
	
	i := 0
	mux := http.NewServeMux()
	for _, streamdef := range config.Resources {
		switch streamdef.Type {
		case "stream":
			log.Printf("Connecting stream %s to %s", streamdef.Serve, streamdef.Remote)
			
			queue := make(chan restreamer.Packet, config.InputBuffer)
			reg := stats.RegisterStream(streamdef.Serve)
			client, err := restreamer.NewClient(streamdef.Remote, queue, config.Timeout, config.Reconnect, reg)
			
			if err == nil {
				client.Connect()
			}
			
			if err == nil {
				clients[streamdef.Serve] = client
				
				streamer := restreamer.NewStreamer(queue, config.OutputBuffer, controller, reg)
				mux.Handle(streamdef.Serve, streamer)
				streamer.Connect()
				
				log.Printf("Handled connection %d", i)
				i++
			} else {
				log.Print(err)
			}
			
		case "static":
			log.Printf("Configuring static resource %s on %s", streamdef.Serve, streamdef.Remote)
			proxy, err := restreamer.NewProxy(streamdef.Remote, config.Timeout, streamdef.Cache, stats)
			if err != nil {
				log.Print(err)
			} else {
				mux.Handle(streamdef.Serve, proxy)
			}
			
		case "api":
			switch streamdef.Api {
			case "health":
				log.Printf("Registering global health API on %s", streamdef.Serve);
				mux.Handle(streamdef.Serve, restreamer.NewHealthApi(stats))
			case "statistics":
				log.Printf("Registering global statistics API on %s", streamdef.Serve);
				mux.Handle(streamdef.Serve, restreamer.NewStatisticsApi(stats))
			case "check":
				log.Printf("Registering stream check API on %s", streamdef.Serve);
				client := clients[streamdef.Remote]
				if client != nil {
					mux.Handle(streamdef.Serve, restreamer.NewStreamStateApi(client))
				} else {
					log.Printf("Error, stream not found: %s", streamdef.Remote);
				}
			default:
				log.Printf("Invalid API type: %s", streamdef.Api);
			}
			
		default:
			log.Printf("Invalid resource type: %s", streamdef.Type);
		}
	}
	
	if i == 0 {
		log.Fatal("No streams available")
	} else {
		log.Print("Starting stats monitor")
		stats.Start()
		log.Print("Starting server")
		log.Fatal(http.ListenAndServe(config.Listen, mux))
	}
}
