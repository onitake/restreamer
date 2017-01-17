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
	"net/http"
	"encoding/json"
	"restreamer"
)

// configuration file structure
// these should be in a JSON dictionary
// in the configuration file
// note that the keys should be lower case
type Configuration struct {
	// the interface to listen on
	Listen string
	// the connection timeout
	// (both input and output)
	Timeout uint
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
	// the maximum number of concurrent connections
	// per stream URL
	MaxConnections uint
	// set to true to disable statistics
	NoStats bool
	// the list of streams
	Streams []struct {
		// the local URL to serve this stream under
		Serve string
		// the upstream URL
		Remote string
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
		stats = restreamer.NewStatistics()
	}
	
	i := 0
	mux := http.NewServeMux()
	for _, streamdef := range config.Streams {
		log.Printf("Connecting stream %s to %s", streamdef.Serve, streamdef.Remote)
		
		queue := make(chan restreamer.Packet, config.InputBuffer)
		reg := stats.RegisterStream(streamdef.Serve, config.MaxConnections)
		
		client, err := restreamer.NewClient(streamdef.Remote, queue, config.Timeout, reg)
		
		if err == nil {
			mux.Handle("/check" + streamdef.Serve, restreamer.NewStreamStateApi(client))
			err = client.Connect()
		}
		
		if err == nil {
			streamer := restreamer.NewStreamer(queue, config.MaxConnections, config.OutputBuffer, reg)
			mux.Handle(streamdef.Serve, streamer)
			streamer.Connect()
			
			log.Printf("Handled connection %d", i)
			i++
		} else {
			log.Print(err)
		}
	}
	
	log.Print("Registering global API endpoints");
	mux.Handle("/health", restreamer.NewHealthApi(stats))
	mux.Handle("/stats", restreamer.NewStatsApi(stats))
	
	if i == 0 {
		log.Fatal("No streams available")
	} else {
		log.Print("Starting stats monitor")
		stats.Start()
		log.Print("Starting server")
		log.Fatal(http.ListenAndServe(config.Listen, mux))
	}
}
