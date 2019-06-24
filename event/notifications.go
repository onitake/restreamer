/* Copyright (c) 2018-2019 Gregor Riepl
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

import "time"

// Notifiable defines the interface for a notification dispatcher.
//
// Inidividual calls cause state changes, which may trigger events.
type Notifiable interface {
	// NotifyConnect reports new connections (if connected is positive) or
	// disconnects (if connected is negative).
	//
	// Connects and disconnects should be reported separately.
	NotifyConnect(connected int)
	// NotifyHeartbeat is called periodically when enabled, to allow sending
	// keepalive messages to a monitoring system
	NotifyHeartbeat(when time.Time)
}
