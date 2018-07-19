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
	"github.com/onitake/restreamer/api"
	"github.com/onitake/restreamer/configuration"
	"github.com/onitake/restreamer/event"
	"github.com/onitake/restreamer/protocol"
	"github.com/onitake/restreamer/streaming"
	"github.com/onitake/restreamer/util"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
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
	errorMainStreamNotFound          = "stream_notfound"
	errorMainInvalidApi              = "invalid_api"
	errorMainInvalidResource         = "invalid_resource"
	errorMainInvalidNotification     = "invalid_notification"
	errorMainMissingNotificationUser = "missing_notification_user"
	errorMainMissingStreamUser       = "missing_stream_user"
	errorMainInvalidAuthentication   = "invalid_authentication"
)

func main() {
	var logbackend util.JsonLogger = &util.ConsoleLogger{}

	logger := &util.ModuleLogger{
		Logger: logbackend,
		Defaults: util.Dict{
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

	config, err := configuration.LoadConfigurationFile(configname)
	if err != nil {
		log.Fatal("Error parsing configuration: ", err)
	}

	logger.Log(util.Dict{
		"event":   eventMainConfig,
		"listen":  config.Listen,
		"timeout": config.Timeout,
	})

	if config.Profile {
		EnableProfiling()
	}

	if config.Log != "" {
		flogger, err := util.NewFileLogger(config.Log, true)
		if err != nil {
			log.Fatal("Error opening log: ", err)
		}
		logbackend = flogger
		logger.Logger = logbackend
	}

	clients := make(map[string]*streaming.Client)

	var stats api.Statistics
	if config.NoStats {
		stats = &api.DummyStatistics{}
	} else {
		stats = api.NewStatistics(config.MaxConnections, config.FullConnections)
	}

	controller := streaming.NewAccessController(config.MaxConnections)
	controller.SetLogger(logbackend)

	queue := event.NewEventQueue(int(config.FullConnections))
	queue.SetLogger(logbackend)
	for _, note := range config.Notifications {
		var typ event.EventType
		switch note.Event {
		case "limit_hit":
			typ = event.EventLimitHit
		case "limit_miss":
			typ = event.EventLimitMiss
		}
		var handler event.Handler
		var err error
		switch note.Type {
		case "url":
			auth := protocol.NewUserAuthenticator(note.Authentication, protocol.NewAuthenticator(note.Authentication, config.UserList))
			if auth == nil {
				logger.Log(util.Dict{
					"event":   eventMainError,
					"error":   errorMainInvalidAuthentication,
					"message": fmt.Sprintf("Invalid authentication configuration, possibly a missing user"),
				})
			}
			urlhandler, err := event.NewUrlHandler(note.Url, auth)
			if err == nil {
				urlhandler.SetLogger(logbackend)
				handler = urlhandler
			}
		}
		if err == nil {
			queue.RegisterEventHandler(typ, handler)
		} else {
			logger.Log(util.Dict{
				"event":   eventMainError,
				"error":   errorMainInvalidNotification,
				"message": fmt.Sprintf("Cannot configure notification: %v", err),
			})
		}
	}
	queue.Start()

	i := 0
	mux := http.NewServeMux()
	for _, streamdef := range config.Resources {
		switch streamdef.Type {
		case "stream":
			logger.Log(util.Dict{
				"event":   eventMainConfigStream,
				"serve":   streamdef.Serve,
				"remote":  streamdef.Remotes,
				"message": fmt.Sprintf("Connecting stream %s to %v", streamdef.Serve, streamdef.Remotes),
			})

			reg := stats.RegisterStream(streamdef.Serve)

			auth := protocol.NewAuthenticator(streamdef.Authentication, config.UserList)

			streamer := streaming.NewStreamer(config.OutputBuffer, controller, auth)
			streamer.SetLogger(logbackend)
			streamer.SetCollector(reg)
			streamer.SetNotifier(queue)

			// shuffle the list here, not later
			// should give a bit more randomness
			remotes := util.ShuffleStrings(rnd, streamdef.Remotes)

			client, err := streaming.NewClient(remotes, streamer, config.Timeout, config.Reconnect, config.ReadTimeout, config.InputBuffer)
			if err == nil {
				client.SetCollector(reg)
				client.SetLogger(logbackend)
				client.Connect()
				clients[streamdef.Serve] = client
				mux.Handle(streamdef.Serve, streamer)

				logger.Log(util.Dict{
					"event":   eventMainHandled,
					"number":  i,
					"message": fmt.Sprintf("Handled connection %d", i),
				})
				i++
			} else {
				log.Print(err)
			}

		case "static":
			logger.Log(util.Dict{
				"event":   eventMainConfigStatic,
				"serve":   streamdef.Serve,
				"remote":  streamdef.Remote,
				"message": fmt.Sprintf("Configuring static resource %s on %s", streamdef.Serve, streamdef.Remote),
			})
			auth := protocol.NewAuthenticator(streamdef.Authentication, config.UserList)
			proxy, err := streaming.NewProxy(streamdef.Remote, config.Timeout, streamdef.Cache, auth)
			if err != nil {
				log.Print(err)
			} else {
				proxy.SetStatistics(stats)
				proxy.SetLogger(logbackend)
				proxy.Start()
				mux.Handle(streamdef.Serve, proxy)
			}

		case "api":
			auth := protocol.NewAuthenticator(streamdef.Authentication, config.UserList)

			switch streamdef.Api {
			case "health":
				logger.Log(util.Dict{
					"event":   eventMainConfigApi,
					"api":     "health",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering global health API on %s", streamdef.Serve),
				})
				mux.Handle(streamdef.Serve, api.NewHealthApi(stats, auth))
			case "statistics":
				logger.Log(util.Dict{
					"event":   eventMainConfigApi,
					"api":     "statistics",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering global statistics API on %s", streamdef.Serve),
				})
				mux.Handle(streamdef.Serve, api.NewStatisticsApi(stats, auth))
			case "check":
				logger.Log(util.Dict{
					"event":   eventMainConfigApi,
					"api":     "check",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering stream check API on %s", streamdef.Serve),
				})
				client := clients[streamdef.Remote]
				if client != nil {
					mux.Handle(streamdef.Serve, api.NewStreamStateApi(client, auth))
				} else {
					logger.Log(util.Dict{
						"event":   eventMainError,
						"error":   errorMainStreamNotFound,
						"api":     "check",
						"remote":  streamdef.Remote,
						"message": fmt.Sprintf("Error, stream not found: %s", streamdef.Remote),
					})
				}
			case "control":
				logger.Log(util.Dict{
					"event":   eventMainConfigApi,
					"api":     "control",
					"serve":   streamdef.Serve,
					"message": fmt.Sprintf("Registering stream control API on %s", streamdef.Serve),
				})
				client := clients[streamdef.Remote]
				if client != nil {
					mux.Handle(streamdef.Serve, api.NewStreamControlApi(client, auth))
				} else {
					logger.Log(util.Dict{
						"event":   eventMainError,
						"error":   errorMainStreamNotFound,
						"api":     "control",
						"remote":  streamdef.Remote,
						"message": fmt.Sprintf("Error, stream not found: %s", streamdef.Remote),
					})
				}
			default:
				logger.Log(util.Dict{
					"event":   eventMainError,
					"error":   errorMainInvalidApi,
					"api":     streamdef.Api,
					"message": fmt.Sprintf("Invalid API type: %s", streamdef.Api),
				})
			}

		default:
			logger.Log(util.Dict{
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
		logger.Log(util.Dict{
			"event":   eventMainStartMonitor,
			"message": "Starting stats monitor",
		})
		stats.Start()
		logger.Log(util.Dict{
			"event":   eventMainStartServer,
			"message": "Starting server",
		})
		log.Fatal(http.ListenAndServe(config.Listen, mux))
	}
}
