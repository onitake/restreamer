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

package streaming

import (
	"fmt"
	"github.com/onitake/restreamer/util"
	"sync"
)

const (
	moduleAcl = "acl"
	//
	eventAclError    = "error"
	eventAclAccepted = "accepted"
	eventAclDenied   = "denied"
	eventAclRemoved  = "removed"
	//
	errorAclNoConnection = "noconnection"
)

// AccessController implements a connection broker that limits
// the maximum number of concurrent connections.
type AccessController struct {
	// maxconnections is a global limit on the number of connections.
	maxconnections uint
	// lock to protect the connection counter
	lock sync.Mutex
	// connections contains the number of active connections.
	// must be accessed atomically.
	connections uint
	// logger is a json logger
	logger *util.ModuleLogger
}

// NewAccessController creates a connection broker object that
// handles access control according to the number of connected clients.
func NewAccessController(maxconnections uint) *AccessController {
	logger := &util.ModuleLogger{
		Logger: &util.ConsoleLogger{},
		Defaults: util.Dict{
			"module": moduleAcl,
		},
		AddTimestamp: true,
	}
	return &AccessController{
		maxconnections: maxconnections,
		logger:         logger,
	}
}

// SetLogger assigns a logger
func (control *AccessController) SetLogger(logger util.JsonLogger) {
	control.logger.Logger = logger
}

// Accept accepts an incoming connection when the maximum number of open connections
// has not been reached yet.
func (control *AccessController) Accept(remoteaddr string, streamer *Streamer) bool {
	accept := false
	// protect concurrent access
	control.lock.Lock()
	// check if the limit is disabled or unreached
	if control.maxconnections == 0 || control.connections < control.maxconnections {
		// and increase the counter
		control.connections++
		accept = true
	}
	control.lock.Unlock()
	// print some info
	if accept {
		control.logger.Log(util.Dict{
			"event":       eventAclAccepted,
			"remote":      remoteaddr,
			"connections": control.connections,
			"max":         control.maxconnections,
			"message":     fmt.Sprintf("Accepted connection from %s, active=%d, max=%d", remoteaddr, control.connections, control.maxconnections),
		})
	} else {
		control.logger.Log(util.Dict{
			"event":       eventAclDenied,
			"remote":      remoteaddr,
			"connections": control.connections,
			"max":         control.maxconnections,
			"message":     fmt.Sprintf("Denied connection from %s, active=%d, max=%d", remoteaddr, control.connections, control.maxconnections),
		})
	}
	// return the result
	return accept
}

// Release decrements the open connections count.
func (control *AccessController) Release(streamer *Streamer) {
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
		control.logger.Log(util.Dict{
			"event":       eventAclRemoved,
			"connections": control.connections,
			"max":         control.maxconnections,
			"message":     fmt.Sprintf("Removed connection, active=%d, max=%d", control.connections, control.maxconnections),
		})
	} else {
		control.logger.Log(util.Dict{
			"event":   eventAclError,
			"error":   errorAclNoConnection,
			"message": fmt.Sprintf("Error, no connection to remove"),
		})
	}
}
