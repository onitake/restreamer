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
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/onitake/restreamer/lib"
)

const (
	moduleMain = "main"
	//
	eventMainError        = "error"
	eventMainConfig       = "config"
	eventMainConfigStream = "stream"
	eventMainConfigStatic = "static"
	eventMainConfigApi    = "api"
	eventMainHandled      = "handled"
	eventMainStartMonitor = "start_monitor"
	eventMainStartServer  = "start_server"
	//
	errorMainStreamNotFound  = "stream_notfound"
	errorMainInvalidApi      = "invalid_api"
	errorMainInvalidResource = "invalid_resource"
)

// Shuffle shuffles a slice using Knuth's version of the Fisher-Yates algorithm.
func ShuffleStrings(rnd *rand.Rand, list []string) []string {
	N := len(list)
	ret := make([]string, N)
	copy(ret, list)
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rnd.Intn(N-i)
		ret[r], ret[i] = ret[i], ret[r]
	}
	return ret
}

func main() {
	var logbackend restreamer.JsonLogger = &restreamer.ConsoleLogger{}

	logger := &restreamer.ModuleLogger{
		Logger: logbackend,
		Defaults: restreamer.Dict{
			"module": moduleMain,
		},
		AddTimestamp: true,
	}

	rnd := rand.New(rand.NewSource(time.Now().Unix()))

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

	logger.Log(restreamer.Dict{
		"event":   eventMainConfig,
		"listen":  config.Listen,
		"timeout": config.Timeout,
	})

	if config.Profile {
		EnableProfiling()
	}

	if config.Log != "" {
		flogger, err := restreamer.NewFileLogger(config.Log, true)
		if err != nil {
			log.Fatal("Error opening log: ", err)
		}
		logbackend = flogger
		logger.Logger = logbackend
	}

	clients := make(map[string]*restreamer.Client)

	var stats restreamer.Statistics
	if config.NoStats {
		stats = &restreamer.DummyStatistics{}
	} else {
		stats = restreamer.NewStatistics(config.MaxConnections)
	}

	controller := restreamer.NewAccessController(config.MaxConnections)
	controller.SetLogger(logbackend)

	i := 0
	mux := http.NewServeMux()
	for _, streamdef := range config.Resources {
		switch streamdef.Type {
		case "stream":
			logger.Log(restreamer.Dict{
				"event":   eventMainConfigStream,
				"serve":   streamdef.Serve,
				"remote":  streamdef.Remote,
				"message": fmt.Sprintf("Connecting stream %s to %s", streamdef.Serve, streamdef.Remote),
			})

			reg := stats.RegisterStream(streamdef.Serve)

			streamer := restreamer.NewStreamer(config.OutputBuffer, controller)
			streamer.SetCollector(reg)
			streamer.SetLogger(logbackend)

			// shuffle the list here, not later
			// should give a bit more randomness
			remotes := ShuffleStrings(rnd, streamdef.Remotes)

			client, err := restreamer.NewClient(remotes, streamer, config.Timeout, config.Reconnect, config.ReadTimeout, config.InputBuffer)
			if err == nil {
				client.SetCollector(reg)
				client.SetLogger(logbackend)
				client.Connect()
				clients[streamdef.Serve] = client
				mux.Handle(streamdef.Serve, streamer)

				logger.Log(restreamer.Dict{
					"event":   eventMainHandled,
					"number":  i,
					"message": fmt.Sprintf("Handled connection %d", i),
				})
				i++
			} else {
				log.Print(err)
			}

		case "static":
			logger.Log(restreamer.Dict{
				"event":   eventMainConfigStatic,
				"serve":   streamdef.Serve,
				"remote":  streamdef.Remote,
				"message": fmt.Sprintf("Configuring static resource %s on %s", streamdef.Serve, streamdef.Remote),
			})
			proxy, err := restreamer.NewProxy(streamdef.Remote, config.Timeout, streamdef.Cache)
			if err != nil {
				log.Print(err)
			} else {
				proxy.SetStatistics(stats)
				proxy.SetLogger(logbackend)
				mux.Handle(streamdef.Serve, proxy)
			}

		case "api":
			switch streamdef.Api {
			case "health":
				logger.Log(restreamer.Dict{
					"event":   eventMainConfigApi,
					"api":     "health",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering global health API on %s", streamdef.Serve),
				})
				mux.Handle(streamdef.Serve, restreamer.NewHealthApi(stats))
			case "statistics":
				logger.Log(restreamer.Dict{
					"event":   eventMainConfigApi,
					"api":     "statistics",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering global statistics API on %s", streamdef.Serve),
				})
				mux.Handle(streamdef.Serve, restreamer.NewStatisticsApi(stats))
			case "check":
				logger.Log(restreamer.Dict{
					"event":   eventMainConfigApi,
					"api":     "check",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering stream check API on %s", streamdef.Serve),
				})
				client := clients[streamdef.Remote]
				if client != nil {
					mux.Handle(streamdef.Serve, restreamer.NewStreamStateApi(client))
				} else {
					logger.Log(restreamer.Dict{
						"event":   eventMainError,
						"error":   errorMainStreamNotFound,
						"api":     "check",
						"remote":  streamdef.Remote,
						"message": fmt.Sprintf("Error, stream not found: %s", streamdef.Remote),
					})
				}
			default:
				logger.Log(restreamer.Dict{
					"event":   eventMainError,
					"error":   errorMainInvalidApi,
					"api":     streamdef.Api,
					"message": fmt.Sprintf("Invalid API type: %s", streamdef.Api),
				})
			}

		default:
			logger.Log(restreamer.Dict{
				"event":   eventMainError,
				"error":   errorMainInvalidResource,
				"type":    streamdef.Type,
				"message": fmt.Sprintf("Invalid resource type: %s", streamdef.Type),
			})
		}
	}

	if i == 0 {
		log.Fatal("No streams available")
	} else {
		logger.Log(restreamer.Dict{
			"event":   eventMainStartMonitor,
			"message": "Starting stats monitor",
		})
		stats.Start()
		logger.Log(restreamer.Dict{
			"event":   eventMainStartServer,
			"message": "Starting server",
		})
		log.Fatal(http.ListenAndServe(config.Listen, mux))
	}
}
