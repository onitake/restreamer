/* Copyright (c) 2019 Gregor Riepl
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

type HeartbeatStopper interface {
	Stop()
}

type Heartbeat struct {
	// ticker fires a heartbeat at regular intervals.
	ticker *time.Ticker
	// target is the notification target.
	// NotifyHeartbeat will be called on each tick.
	target Notifiable
}

// NewHeartbeat creates a new heartbeat ticker.
//
// On each heartbeat, target.NotifyHeartbeat will be called with the current timestamp.
// Note that this happens asynchronously from a separate goroutine.
func NewHeartbeat(interval time.Duration, target Notifiable) *Heartbeat {
	heartbeat := &Heartbeat{
		ticker: time.NewTicker(interval),
		target: target,
	}
	go heartbeat.loop()
	return heartbeat
}

// loop is the ticker run loop
func (heartbeat *Heartbeat) loop() {
	logger.Logkv(
		"event", queueEventHeartbeatStart,
		"message", "Starting heartbeat goroutine",
	)
	// process events (also drains when the channel is closed)
	for range heartbeat.ticker.C {
		logger.Logkv(
			"event", queueEventHeartbeatFire,
			"message", "Firing heartbeat",
		)
		heartbeat.target.NotifyHeartbeat(time.Now())
	}
	logger.Logkv(
		"event", queueEventHeartbeatStop,
		"message", "Stopping heartbeat goroutine",
	)
}

func (heartbeat *Heartbeat) Stop() {
	heartbeat.ticker.Stop()
}

type DummyHeartbeat struct{}

// Stop does nothing
func (*DummyHeartbeat) Stop() {
	// do nothing
}
