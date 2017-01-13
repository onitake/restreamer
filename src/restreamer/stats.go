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
	connections int64
	packetsReceived uint64
	packetsSent uint64
	packetsDropped uint64
	connected int32
}

// Notify that a new connection was added
func (stats *CurrentStreamStatistics) ConnectionAdded() {
	atomic.AddInt64(&stats.connections, 1)
}

// Notify that a connection was removed
func (stats *CurrentStreamStatistics) ConnectionRemoved() {
	atomic.AddInt64(&stats.connections, -1)
}

// Notify that a packet was received
func (stats *CurrentStreamStatistics) PacketReceived() {
	atomic.AddUint64(&stats.packetsReceived, 1)
}

// Notify that a packet was sent
func (stats *CurrentStreamStatistics) PacketSent() {
	atomic.AddUint64(&stats.packetsSent, 1)
}

// Notify that a packet was dropped
func (stats *CurrentStreamStatistics) PacketDropped() {
	atomic.AddUint64(&stats.packetsDropped, 1)
}

// Notify that upstream is live
func (stats *CurrentStreamStatistics) Connect() {
	atomic.StoreInt32(&stats.connected, 1)
}

// Notify that upstream is offline
func (stats *CurrentStreamStatistics) Disconnect() {
	atomic.StoreInt32(&stats.connected, 0)
}

// Notify that upstream is offline
func (stats *CurrentStreamStatistics) IsConnected() bool {
	return atomic.LoadInt32(&stats.connected) != 0
}

// StreamStatistics is the current state of a single stream
// or all streams combined.
// Updates are collected and evaluated periodically.
// Connected status can be checked by calling Connected().
type StreamStatistics struct {
	Connections int64
	MaxConnections int64
	TotalConnections int64
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
func (stats *Statistics) update(delta time.Duration) {
	// acquire the global write lock
	stats.lock.Lock()
	
	// reset global counters temporarily
	stats.global.Connections = 0
	stats.global.MaxConnections = 0
	stats.global.TotalConnections = 0
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
	for name, internal := range stats.internal {
		stream := stats.streams[name]
		
		// fetch data from the atomic counters
		connections := atomic.LoadInt64(&internal.connections)
		packetsReceived := atomic.LoadUint64(&internal.packetsReceived)
		packetsSent := atomic.LoadUint64(&internal.packetsSent)
		packetsDropped := atomic.LoadUint64(&internal.packetsDropped)
		connected := internal.IsConnected()
		
		// calculate the updated values
		bytesReceived := packetsReceived * PacketSize
		bytesSent := packetsSent * PacketSize
		bytesDropped := packetsDropped * PacketSize
		packetsPerSecondReceived := float64(packetsReceived) / delta.Seconds()
		packetsPerSecondSent := float64(packetsSent) / delta.Seconds()
		packetsPerSecondDropped := float64(packetsDropped) / delta.Seconds()
		bytesPerSecondReceived := float64(bytesReceived) / delta.Seconds()
		bytesPerSecondSent := float64(bytesSent) / delta.Seconds()
		bytesPerSecondDropped := float64(bytesDropped) / delta.Seconds()
		
		// and assign them to the stream stats
		stream.Connections = connections
		stream.TotalConnections += connections
		stream.TotalPacketsReceived = packetsReceived
		stream.TotalPacketsSent = packetsSent
		stream.TotalPacketsDropped = packetsDropped
		stream.TotalBytesReceived += bytesReceived
		stream.TotalBytesSent += bytesSent
		stream.TotalBytesDropped += bytesDropped
		stream.PacketsPerSecondReceived = uint64(packetsPerSecondReceived)
		stream.PacketsPerSecondSent = uint64(packetsPerSecondSent)
		stream.PacketsPerSecondDropped = uint64(packetsPerSecondDropped)
		stream.BytesPerSecondReceived = uint64(bytesPerSecondReceived)
		stream.BytesPerSecondSent = uint64(bytesPerSecondSent)
		stream.BytesPerSecondDropped = uint64(bytesPerSecondDropped)
		stream.Connected = connected
		
		// update the global counters as well
		stats.global.Connections += connections
		stats.global.MaxConnections += stream.MaxConnections
		stats.global.TotalConnections += stream.TotalConnections
		stats.global.TotalPacketsReceived += packetsReceived
		stats.global.TotalPacketsSent += packetsSent
		stats.global.TotalPacketsDropped += packetsDropped
		stats.global.TotalBytesReceived += bytesReceived
		stats.global.TotalBytesSent += bytesSent
		stats.global.TotalBytesDropped += bytesDropped
		stats.global.PacketsPerSecondReceived += uint64(packetsPerSecondReceived)
		stats.global.PacketsPerSecondSent += uint64(packetsPerSecondSent)
		stats.global.PacketsPerSecondDropped += uint64(packetsPerSecondDropped)
		stats.global.BytesPerSecondReceived += uint64(bytesPerSecondReceived)
		stats.global.BytesPerSecondSent += uint64(bytesPerSecondSent)
		stats.global.BytesPerSecondDropped += uint64(bytesPerSecondDropped)
		if connected {
			stats.global.Connected = true
		}
	}
	
	// and done
	stats.lock.Unlock()
}

// loop runs a ticker to update all statistics periodically.
func (stats *Statistics) loop() {
	running := true
	// TODO make the interval configurable
	ticker := time.NewTicker(5 * time.Second)
	// pre-init
	before := time.Now()
	for running {
		select {
			case <-stats.shutdown:
				running = false
			case <-ticker.C:
				// calculate the elapsed time and update
				now := time.Now()
				stats.update(now.Sub(before))
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
