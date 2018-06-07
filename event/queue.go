/* Copyright (c) 2018 Gregor Riepl
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

package event

import (
	"fmt"
	"math"
	"sync"
	"github.com/onitake/restreamer/util"
)

const (
	moduleQueue = "queue"
	//
	queueEventError = "error"
	queueEventLimitHit   = "hit"
	queueEventLimitMiss  = "miss"
	queueEventStarting = "starting"
	queueEventStopping = "stopping"
	queueEventStarted = "started"
	queueEventReceived = "received"
	queueEventDraining = "draining"
	queueEventStopped = "stopped"
	queueEventConnect = "connect"
	//
	queueErrorAlreadyRunning = "already_running"
	queueErrorInvalidNotification = "invalid_notification"
	queueErrorUnderflow = "underflow"
	queueErrorOverflow = "overflow"
	queueErrorRegister = "register"
	queueErrorNotRegistered = "not_registered"
)

const (
	// queueSize is the maximum number of notifications to enqueue before we block
	queueSize int = 10
)

// changeType enumerates all possible state change notifications
type changeType int
const (
	changeConnect changeType = iota
)

// stateChange encapsulates a state change notification
type stateChange struct {
	// typ contains the notification type
	typ changeType
	// connected contains the number of new connections.
	// Can be negative if connections are dropped.
	connected int
}

// EventQueue encapsulates state for a connection load reporting callback.
//
// The hit/miss pairs define a hysteresis range to avoid "flapping" reports
// when the number of connections changes quickly around a limit.
type EventQueue struct {
	// limit sets the number of connections when a hit is reported
	limit int
	// handlers contains all event handlers
	handlers map[EventType]map[Handler]bool
	// internal notification channel for the reporting thread
	notifier chan *stateChange
	// connections contains the number of active connections.
	// only accessed from the reporting thread
	connections int
	// shutdown is the internal shutdown notifier
	shutdown chan struct{}
	// running tells if the notifier is currently active
	running bool
	// waiter allows waiting for shutdown
	waiter *sync.WaitGroup
	// running represents the current running state of the notifier
	// logger is a json logger
	logger *util.ModuleLogger
}

// Creates a new connection load report notifier.
//
// limit specifies the reporting threshold.
func NewEventQueue(limit int) *EventQueue {
	if limit < 0 {
		panic("limit is out of range")
	}
	logger := &util.ModuleLogger{
		Logger: &util.ConsoleLogger{},
		Defaults: util.Dict{
			"module": moduleQueue,
		},
		AddTimestamp: true,
	}
	return &EventQueue{
		limit: limit,
		handlers: make(map[EventType]map[Handler]bool),
		logger:         logger,
		waiter: &sync.WaitGroup{},
	}
}

// SetLogger assigns a logger
func (reporter *EventQueue) SetLogger(logger util.JsonLogger) {
	reporter.logger.Logger = logger
}

// Start launches the reporting goroutine.
//
// To stop the reporter, call Shutdown().
func (reporter *EventQueue) Start() {
	reporter.logger.Log(util.Dict{
		"event":       "check_start",
		"message":     "Checking if the handler can be started",
	})
	// check if we're running already
	if !reporter.running {
		reporter.logger.Log(util.Dict{
			"event":       queueEventStarting,
			"message":     "Starting notification handler",
		})
		// initialise the channels
		reporter.shutdown = make(chan struct{})
		reporter.notifier = make(chan *stateChange, queueSize)
		// set running state
		reporter.running = true
		reporter.waiter.Add(1)
		// and start the handler
		go reporter.run()
	} else {
		reporter.logger.Log(util.Dict{
			"event":       queueEventError,
			"error":       queueErrorAlreadyRunning,
			"message":     "Notification handler already running, won't start again",
		})
	}
}

// Shutdown stops the load reporter and waits for completion.
//
// You must not send any notifications after calling this method.
func (reporter *EventQueue) Shutdown() {
	reporter.logger.Log(util.Dict{
		"event":       queueEventStopping,
		"message":     "Stopping notification handler",
	})
	// signal shutdown
	if reporter.running {
		close(reporter.shutdown)
		reporter.waiter.Wait()
	}
}

// run is the notification handling loop
func (reporter *EventQueue) run() {
	reporter.logger.Log(util.Dict{
		"event":       queueEventStarted,
		"message":     "Notification handler started",
	})
	running := true
	for running {
		select {
			case <-reporter.shutdown:
				running = false
			case message := <-reporter.notifier:
				reporter.handle(message)
		}
	}
	reporter.logger.Log(util.Dict{
		"event":       queueEventDraining,
		"message":     "Draining notification queue",
	})
	// drain the notification channel and close it
	close(reporter.notifier)
	for range reporter.notifier {}
	reporter.logger.Log(util.Dict{
		"event":       queueEventStopped,
		"message":     "Stopped notification handler",
	})
	// and we're done
	reporter.running = false
	reporter.waiter.Done()
}

// handle handles a single message
func (reporter *EventQueue) handle(message *stateChange) {
	switch message.typ {
		case changeConnect:
			reporter.handleConnect(message.connected)
		default:
			reporter.logger.Log(util.Dict{
				"event":       queueEventError,
				"error":     queueErrorInvalidNotification,
				"type":        message.typ,
			})
	}
}

// handleConnect handles a connected clients state change
func (reporter *EventQueue) handleConnect(connected int) {
	reporter.logger.Log(util.Dict{
		"event":       queueEventConnect,
		"message":     fmt.Sprintf("Number of connections changed by %d, new number of connections: %d", connected, reporter.connections),
		"connected": connected,
		"connections": reporter.connections,
	})
	// calculate the new connection count
	var newconn int
	if connected < 0 && -connected > reporter.connections {
		reporter.logger.Log(util.Dict{
			"event":       queueEventError,
			"error":       queueErrorUnderflow,
			"message":     "Number of disconnects exceeds number of connections, setting to 0",
			"connected": connected,
			"connections": reporter.connections,
		})
		newconn = 0
	} else if connected > math.MaxUint32 - reporter.connections {
		reporter.logger.Log(util.Dict{
			"event":       queueEventError,
			"error":       queueErrorOverflow,
			"message":     "Number of connects exceeds counter range, clamping to limit",
			"connected": connected,
			"connections": reporter.connections,
		})
		newconn = math.MaxUint32
	} else {
		newconn = reporter.connections + connected
	}
	// check if the limit is enabled
	if reporter.limit != 0 {
		// handle state transitions
		if reporter.connections >= reporter.limit {
			if newconn < reporter.limit {
				// hit -> miss
				reporter.logger.Log(util.Dict{
					"event":       queueEventLimitMiss,
					"message":     "Limit missed",
					"connections": reporter.connections,
					"new": newconn,
					"limit": reporter.limit,
				})
				for handler, ok := range reporter.handlers[EventLimitMiss] {
					if ok {
						handler.HandleEvent(EventLimitMiss, reporter.connections, newconn, reporter.limit)
					}
				}
			}
		} else {
			if newconn >= reporter.limit {
				// miss -> hit
				reporter.logger.Log(util.Dict{
					"event":       queueEventLimitHit,
					"message":     "Limit hit",
					"connections": reporter.connections,
					"new": newconn,
					"limit": reporter.limit,
				})
				for handler, ok := range reporter.handlers[EventLimitHit] {
					if ok {
						handler.HandleEvent(EventLimitHit, reporter.connections, newconn, reporter.limit)
					}
				}
			}
		}
	}
	// update the counter
	reporter.connections = newconn
}

func (reporter *EventQueue) RegisterEventHandler(typ EventType, handler Handler) {
	if reporter.running {
		reporter.logger.Log(util.Dict{
			"event":       queueEventError,
			"error":       queueErrorRegister,
			"message":     "Cannot register new handlers while the queue is running",
		})
	} else {
		if _, ok := reporter.handlers[typ]; !ok {
			reporter.handlers[typ] = make(map[Handler]bool)
		}
		reporter.handlers[typ][handler] = true
	}
}

func (reporter *EventQueue) UnregisterEventHandler(typ EventType, handler Handler) {
	if reporter.running {
		reporter.logger.Log(util.Dict{
			"event":       queueEventError,
			"error":       queueErrorRegister,
			"message":     "Cannot unregister new handlers while the queue is running",
		})
	} else {
		if _, ok := reporter.handlers[typ][handler]; ok {
			delete(reporter.handlers[typ], handler)
		} else {
			reporter.logger.Log(util.Dict{
				"event":       queueEventError,
				"error":       queueErrorNotRegistered,
				"message":     "Event handler wasn't registered",
			})
		}
	}
}

func (reporter *EventQueue) NotifyConnect(connected int) {
	// construct the notification message and pass it down the queue
	message := &stateChange{
		typ: changeConnect,
		connected: connected,
	}
	reporter.notifier<- message
}
