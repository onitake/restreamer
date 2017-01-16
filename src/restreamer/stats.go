/* Copyright (c) 2017 Gregor Riepl
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
	"time"
	"sync"
	"sync/atomic"
)

// CurrentStreamStatistics represents per-stream state information
// and is continuously updated by the corresponding streamer.
// Use the provided accessor methods for this purpose.
type CurrentStreamStatistics struct {
	// total number of connections
	connections int64
	// total number of received packets
	packetsReceived uint64
	// total number of sent packets
	packetsSent uint64
	// total number of dropped packets
	packetsDropped uint64
	// upstream connection state, 0 = offline, !0 = connected
	connected int32
}

// ConnectionAdded notifies that a new downstream client connected.
func (stats *CurrentStreamStatistics) ConnectionAdded() {
	atomic.AddInt64(&stats.connections, 1)
}

// ConnectionRemoved notifies that a downstream client disconnected.
func (stats *CurrentStreamStatistics) ConnectionRemoved() {
	atomic.AddInt64(&stats.connections, -1)
}

// PacketReceived notifies that a packet was received.
func (stats *CurrentStreamStatistics) PacketReceived() {
	atomic.AddUint64(&stats.packetsReceived, 1)
}

// PacketReceived notifies that a packet was sent.
func (stats *CurrentStreamStatistics) PacketSent() {
	atomic.AddUint64(&stats.packetsSent, 1)
}

// PacketReceived notifies that a packet was dropped.
func (stats *CurrentStreamStatistics) PacketDropped() {
	atomic.AddUint64(&stats.packetsDropped, 1)
}

// SourceConnected notifies that upstream is live.
func (stats *CurrentStreamStatistics) SourceConnected() {
	atomic.StoreInt32(&stats.connected, 1)
}

// SourceDisconnected notifies that upstream is offline.
func (stats *CurrentStreamStatistics) SourceDisconnected() {
	atomic.StoreInt32(&stats.connected, 0)
}

// IsUpstreamConnected tells you if upstream is connected.
func (stats *CurrentStreamStatistics) IsUpstreamConnected() bool {
	return atomic.LoadInt32(&stats.connected) != 0
}

// clone creates a copy of the stats object - useful for
// storing state temporarily.
func (stats *CurrentStreamStatistics) clone() *CurrentStreamStatistics {
	return &CurrentStreamStatistics{
		connections: atomic.LoadInt64(&stats.connections),
		packetsReceived: atomic.LoadUint64(&stats.packetsReceived),
		packetsSent: atomic.LoadUint64(&stats.packetsSent),
		packetsDropped: atomic.LoadUint64(&stats.packetsDropped),
		connected: atomic.LoadInt32(&stats.connected),
	}
}

// invsub subtracts this stats object from another and sets each
// value to the difference. Note: Should not be used on atomic values
// directly. clone() first.
// "connected" is copied directly from "to".
// Useful if you want to calculate a delta, then replace the previous
// value with the current one:
// prev := CurrentStreamStatistics{}
// for {
//   current := CurrentStreamStatistics{}
//   prev.invsub(current)
//   doSomethingWithPrev(prev)
//   prev = current
// }
func (from *CurrentStreamStatistics) invsub(to *CurrentStreamStatistics) {
	from.connections = to.connections - from.connections
	from.packetsReceived = to.packetsReceived - from.packetsReceived
	from.packetsSent= to.packetsSent - from.packetsSent
	from.packetsDropped= to.packetsDropped - from.packetsDropped
	from.connected = to.connected
}

// StreamStatistics is the current state of a single stream
// or all streams combined.
// Updates are collected and evaluated periodically.
// Connected status can be checked by calling Connected().
type StreamStatistics struct {
	Connections int64
	MaxConnections int64
	TotalPacketsReceived uint64
	TotalPacketsSent uint64
	TotalPacketsDropped uint64
	TotalBytesReceived uint64
	TotalBytesSent uint64
	TotalBytesDropped uint64
	PacketsPerSecondReceived uint64
	PacketsPerSecondSent uint64
	PacketsPerSecondDropped uint64
	BytesPerSecondReceived uint64
	BytesPerSecondSent uint64
	BytesPerSecondDropped uint64
	Connected bool
}

// Statistics holds system and stream status information.
// Streams update their state continuously, but data fields are only updated in periodic intervals.
// There is also an HTTP/JSON API facility available through the New...Api() methods in api.go.
type Statistics struct {
	lock sync.RWMutex
	running bool
	shutdown chan bool
	internal map[string]*CurrentStreamStatistics
	streams map[string]*StreamStatistics
	global *StreamStatistics
}

// update updates the aggregated statistics from the current state of each stream.
func (stats *Statistics) update(delta time.Duration, change map[string]*CurrentStreamStatistics) {
	// acquire the global write lock
	stats.lock.Lock()
	
	// reset the global counters
	stats.global.Connections = 0
	stats.global.MaxConnections = 0
	stats.global.TotalPacketsReceived = 0
	stats.global.TotalPacketsSent = 0
	stats.global.TotalPacketsDropped = 0
	stats.global.TotalBytesReceived = 0
	stats.global.TotalBytesSent = 0
	stats.global.TotalBytesDropped = 0
	stats.global.PacketsPerSecondReceived = 0
	stats.global.PacketsPerSecondSent = 0
	stats.global.PacketsPerSecondDropped = 0
	stats.global.BytesPerSecondReceived = 0
	stats.global.BytesPerSecondSent = 0
	stats.global.BytesPerSecondDropped = 0
	stats.global.Connected = false
	
	// loop over all streams
	for name, stream := range stats.streams {
		diff := change[name]
		
		// update the stats
		stream.Connections += diff.connections
		stream.TotalPacketsReceived += diff.packetsReceived
		stream.TotalPacketsSent += diff.packetsSent
		stream.TotalPacketsDropped += diff.packetsDropped
		stream.TotalBytesReceived = stream.TotalPacketsReceived * PacketSize
		stream.TotalBytesSent = stream.TotalPacketsSent * PacketSize
		stream.TotalBytesDropped = stream.TotalPacketsDropped * PacketSize
		stream.PacketsPerSecondReceived = uint64(float64(diff.packetsReceived) / delta.Seconds())
		stream.PacketsPerSecondSent = uint64(float64(diff.packetsSent) / delta.Seconds())
		stream.PacketsPerSecondDropped = uint64(float64(diff.packetsDropped) / delta.Seconds())
		stream.BytesPerSecondReceived = stream.PacketsPerSecondReceived * PacketSize
		stream.BytesPerSecondSent = stream.PacketsPerSecondSent * PacketSize
		stream.BytesPerSecondDropped = stream.PacketsPerSecondDropped * PacketSize
		stream.Connected = diff.connected != 0
		
		// update the global counters as well
		stats.global.Connections += stream.Connections
		stats.global.MaxConnections += stream.MaxConnections
		stats.global.TotalPacketsReceived += stream.TotalPacketsReceived
		stats.global.TotalPacketsSent += stream.TotalPacketsSent
		stats.global.TotalPacketsDropped += stream.TotalPacketsDropped
		stats.global.TotalBytesReceived += stream.TotalBytesReceived
		stats.global.TotalBytesSent += stream.TotalBytesSent
		stats.global.TotalBytesDropped += stream.TotalBytesDropped
		stats.global.PacketsPerSecondReceived += stream.PacketsPerSecondReceived
		stats.global.PacketsPerSecondSent += stream.PacketsPerSecondSent
		stats.global.PacketsPerSecondDropped += stream.PacketsPerSecondDropped
		stats.global.BytesPerSecondReceived += stream.BytesPerSecondReceived
		stats.global.BytesPerSecondSent += stream.BytesPerSecondSent
		stats.global.BytesPerSecondDropped += stream.BytesPerSecondDropped
		if stream.Connected {
			stats.global.Connected = true
		}
	}
	
	// and done
	stats.lock.Unlock()
}

// delta calculates the difference between a previous internal state
// and the current state and returns a copy of the current state.
// The previous state (the argument) is replaced with the difference.
func (stats *Statistics) delta(previous map[string]*CurrentStreamStatistics) map[string]*CurrentStreamStatistics {
	stats.lock.RLock()
	current := make(map[string]*CurrentStreamStatistics)
	for name, stream := range stats.internal {
		update := stream.clone()
		previous[name].invsub(update)
		current[name] = update
	}
	stats.lock.RUnlock()
	return current
}

// loop runs a ticker to update all statistics periodically.
func (stats *Statistics) loop() {
	running := true
	// TODO make the interval configurable
	ticker := time.NewTicker(1 * time.Second)

	// pre-init - store the current time and state
	before := time.Now()
	stats.lock.RLock()
	previous := make(map[string]*CurrentStreamStatistics)
	for name, stream := range stats.internal {
		previous[name] = stream.clone()
	}
	stats.lock.RUnlock()

	for running {
		select {
			case <-stats.shutdown:
				running = false
			case <-ticker.C:
				// calculate the elapsed time
				now := time.Now()
				// calculate the state delta and update the stored state
				delta := previous
				previous = stats.delta(previous)
				// and update
				stats.update(now.Sub(before), delta)
				// stash the current time
				before = now
		}
	}
	// this should close the channel as well
	ticker.Stop()
	stats.running = false
}

// Start starts the updater thread.
func (stats *Statistics) Start() {
	if !stats.running {
		stats.running = true
		go stats.loop()
	}
}

// Stop stops the updater thread.
func (stats *Statistics) Stop() {
	if stats.running {
		stats.shutdown<- true
	}
}

// NewStatistics creates a new statistics container.
// You can start and stop the periodic updater using Start() and Stop().
// Register your streams with RegisterStream(), this will return an updateable
// statistics object. You should not write to the individual fields directly,
// instead access them using the Add...() methods.
// Snapshots of the aggregated statistics can then be means of the Get...() methods.
func NewStatistics() (*Statistics) {
	stats := &Statistics{
		shutdown: make(chan bool),
		internal: make(map[string]*CurrentStreamStatistics),
		streams: make(map[string]*StreamStatistics),
		global: &StreamStatistics{},
	}
	return stats
}

// RegisterStream adds a new stream to the map.
// The name will be used as the lookup key, and maxconns is the maximum number of allowed connections.
func (stats *Statistics) RegisterStream(name string, maxconns uint) *CurrentStreamStatistics {
	current := &CurrentStreamStatistics{}
	stats.lock.Lock()
	stats.internal[name] = current
	stats.streams[name] = &StreamStatistics{
		MaxConnections: int64(maxconns),
	}
	stats.lock.Unlock()
	return current
}

// RemoveStream removes a stream from the map.
func (stats *Statistics) RemoveStream(name string) {
	stats.lock.Lock()
	delete(stats.internal, name)
	delete(stats.streams, name)
	stats.lock.Unlock()
}

// GetStreamStatistics fetches the statistics for a stream.
// The returned object is a copy does not need to be handled with care.
func (stats *Statistics) GetStreamStatistics(name string) *StreamStatistics {
	stats.lock.RLock()
	stream := *stats.streams[name]
	stats.lock.RUnlock()
	return &stream
}

// GetAllStreamStatistics fetches the statistics for all streams.
// The returned object is a copy does not need to be handled with care.
func (stats *Statistics) GetAllStreamStatistics() map[string]*StreamStatistics {
	stats.lock.RLock()
	streams := make(map[string]*StreamStatistics, len(stats.streams))
	for name, stream := range stats.streams {
		scopy := *stream
		streams[name] = &scopy
	}
	stats.lock.RUnlock()
	return streams
}

// GetGlobalStatistics fetches the global statistics.
// The returned object is a copy does not need to be handled with care.
func (stats *Statistics) GetGlobalStatistics() *StreamStatistics {
	stats.lock.RLock()
	global := *stats.global
	stats.lock.RUnlock()
	return &global
}
