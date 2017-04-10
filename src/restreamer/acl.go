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

package restreamer

import (
	"log"
	"sync"
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
}

// NewAccessController creates a connection broker object that
// handles access control according to the number of connected clients.
func NewAccessController(maxconnections uint) *AccessController {
	return &AccessController{
		maxconnections: maxconnections,
	}
}

// Accept accepts an incoming connection when the maximum number of open connections
// has not been reached yet.
func (control *AccessController) Accept(remoteaddr string, streamer *Streamer) bool {
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
		log.Printf("Accepted connection from %s @%p, active=%d, max=%d\n", remoteaddr, streamer, control.connections, control.maxconnections)
	} else {
		log.Printf("Denied connection from %s @%p, active=%d, max=%d\n", remoteaddr, streamer, control.connections, control.maxconnections)
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
		log.Printf("Removed connection @%p\n", streamer)
	} else {
		log.Printf("Error, no connection to remove @%p\n", streamer)
	}
}
