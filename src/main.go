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
	"restreamer"
)

func main() {
	var configname string
	if len(os.Args) > 1 {
		configname = os.Args[1]
	} else {
		configname = "restreamer.json"
	}
	
	config, err := restreamer.LoadConfiguration(configname)
	if err != nil {
		log.Fatal("Error parsing configuration: ", err)
	}

	log.Printf("Listen = %s", config.Listen)
	log.Printf("Timeout = %d", config.Timeout)
	
	var stats restreamer.Statistics
	if config.NoStats {
		stats = &restreamer.DummyStatistics{}
	} else {
		stats = restreamer.NewStatistics(config.MaxConnections)
	}
	
	controller := restreamer.NewAccessController(config.MaxConnections)
	
	var logger restreamer.JsonLogger
	if config.Log == "" {
		logger = &restreamer.DummyLogger{}
	} else {
		logger = restreamer.NewFileLogger(config.Log, true)
	}
	
	clients := make(map[string]*restreamer.Client)
	
	i := 0
	mux := http.NewServeMux()
	for _, streamdef := range config.Resources {
		switch streamdef.Type {
		case "stream":
			log.Printf("Connecting stream %s to %s", streamdef.Serve, streamdef.Remote)
			
			queue := make(chan restreamer.Packet, config.InputBuffer)
			reg := stats.RegisterStream(streamdef.Serve)

			client, err := restreamer.NewClient(streamdef.Remotes, queue, config.Timeout, config.Reconnect)
			if err == nil {
				client.SetCollector(reg)
				client.SetLogger(logger)
				client.Connect()
			}
			
			if err == nil {
				clients[streamdef.Serve] = client
				
				streamer := restreamer.NewStreamer(queue, config.OutputBuffer, controller)
				streamer.SetCollector(reg)
				client.SetLogger(logger)
				mux.Handle(streamdef.Serve, streamer)
				streamer.Connect()
				
				log.Printf("Handled connection %d", i)
				i++
			} else {
				log.Print(err)
			}
			
		case "static":
			log.Printf("Configuring static resource %s on %s", streamdef.Serve, streamdef.Remote)
			proxy, err := restreamer.NewProxy(streamdef.Remote, config.Timeout, streamdef.Cache)
			if err != nil {
				log.Print(err)
			} else {
				proxy.SetStatistics(stats)
				proxy.SetLogger(logger)
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
