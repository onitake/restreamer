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

// Collector is the public face of a statistics collector.
// It is implemented by the individual stream stats.
type Collector interface {
	// ConnectionAdded notifies that a new downstream client connected.
	ConnectionAdded()
	// ConnectionRemoved notifies that a downstream client disconnected.
	ConnectionRemoved()
	// PacketReceived notifies that a packet was received.
	PacketReceived()
	// PacketReceived notifies that a packet was sent.
	PacketSent()
	// PacketReceived notifies that a packet was dropped.
	PacketDropped()
	// SourceConnected notifies that upstream is live.
	SourceConnected()
	// SourceDisconnected notifies that upstream is offline.
	SourceDisconnected()
	// IsUpstreamConnected tells you if upstream is connected.
	IsUpstreamConnected() bool
}

// realCollector represents per-stream state information
// and is continuously updated by the corresponding streamer.
// Use the provided accessor methods for this purpose.
type realCollector struct {
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

func (stats *realCollector) ConnectionAdded() {
	atomic.AddInt64(&stats.connections, 1)
}

func (stats *realCollector) ConnectionRemoved() {
	atomic.AddInt64(&stats.connections, -1)
}

func (stats *realCollector) PacketReceived() {
	atomic.AddUint64(&stats.packetsReceived, 1)
}

func (stats *realCollector) PacketSent() {
	atomic.AddUint64(&stats.packetsSent, 1)
}

func (stats *realCollector) PacketDropped() {
	atomic.AddUint64(&stats.packetsDropped, 1)
}

func (stats *realCollector) SourceConnected() {
	atomic.StoreInt32(&stats.connected, 1)
}

func (stats *realCollector) SourceDisconnected() {
	atomic.StoreInt32(&stats.connected, 0)
}

func (stats *realCollector) IsUpstreamConnected() bool {
	return atomic.LoadInt32(&stats.connected) != 0
}

// clone creates a copy of the stats object - useful for
// storing state temporarily.
func (stats *realCollector) clone() *realCollector {
	return &realCollector{
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
// prev := realCollector{}
// for {
//   current := realCollector{}
//   prev.invsub(current)
//   doSomethingWithPrev(prev)
//   prev = current
// }
func (from *realCollector) invsub(to *realCollector) {
	from.connections = to.connections - from.connections
	from.packetsReceived = to.packetsReceived - from.packetsReceived
	from.packetsSent= to.packetsSent - from.packetsSent
	from.packetsDropped= to.packetsDropped - from.packetsDropped
	from.connected = to.connected
}

// StreamStatistics is the current state of a single stream
// or all streams combined.
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

// Statistics is the access interface for a stat tracker.
// Streams update their state continuously, but data fields are only updated in periodic intervals.
// There is also an HTTP/JSON API facility available through the New...Api() methods in api.go.
type Statistics interface {
	// Start starts the updater thread.
	Start()
	// Stop stops the updater thread.
	Stop()
	// RegisterStream adds a new stream to the map.
	// The name will be used as the lookup key.
	RegisterStream(name string) Collector
	// RemoveStream removes a stream from the map.
	RemoveStream(name string)
	// GetStreamStatistics fetches the statistics for a stream.
	// The returned object is a copy does not need to be handled with care.
	GetStreamStatistics(name string) *StreamStatistics
	// GetAllStreamStatistics fetches the statistics for all streams.
	// The returned object is a copy does not need to be handled with care.
	GetAllStreamStatistics() map[string]*StreamStatistics
	// GetGlobalStatistics fetches the global statistics.
	// The returned object is a copy does not need to be handled with care.
	GetGlobalStatistics() *StreamStatistics
}

// realStatistics implements a full statistics collector and API endpoint generator.
type realStatistics struct {
	lock sync.RWMutex
	running bool
	shutdown chan bool
	internal map[string]*realCollector
	streams map[string]*StreamStatistics
	global *StreamStatistics
}

// NewStatistics creates a new statistics container.
// You can start and stop the periodic updater using Start() and Stop().
// Register your streams with RegisterStream(), this will return an updateable
// statistics object. You should not write to the individual fields directly,
// instead access them using the Add...() methods.
// Snapshots of the aggregated statistics can then be means of the Get...() methods.
func NewStatistics(maxconns uint) Statistics {
	stats := &realStatistics{
		shutdown: make(chan bool),
		internal: make(map[string]*realCollector),
		streams: make(map[string]*StreamStatistics),
		global: &StreamStatistics{
			MaxConnections: int64(maxconns),
		},
	}
	return stats
}

// update updates the aggregated statistics from the current state of each stream.
func (stats *realStatistics) update(delta time.Duration, change map[string]*realCollector) {
	// acquire the global write lock
	stats.lock.Lock()
	
	// reset the global counters
	stats.global.Connections = 0
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
func (stats *realStatistics) delta(previous map[string]*realCollector) map[string]*realCollector {
	stats.lock.RLock()
	current := make(map[string]*realCollector)
	for name, stream := range stats.internal {
		update := stream.clone()
		previous[name].invsub(update)
		current[name] = update
	}
	stats.lock.RUnlock()
	return current
}

// loop runs a ticker to update all statistics periodically.
func (stats *realStatistics) loop() {
	running := true
	// TODO make the interval configurable
	ticker := time.NewTicker(1 * time.Second)

	// pre-init - store the current time and state
	before := time.Now()
	stats.lock.RLock()
	previous := make(map[string]*realCollector)
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
func (stats *realStatistics) Start() {
	if !stats.running {
		stats.running = true
		go stats.loop()
	}
}

// Stop stops the updater thread.
func (stats *realStatistics) Stop() {
	if stats.running {
		stats.shutdown<- true
	}
}

// RegisterStream adds a new stream to the map.
// The name will be used as the lookup key.
func (stats *realStatistics) RegisterStream(name string) Collector {
	current := &realCollector{}
	stats.lock.Lock()
	stats.internal[name] = current
	stats.streams[name] = &StreamStatistics{}
	stats.lock.Unlock()
	return current
}

// RemoveStream removes a stream from the map.
func (stats *realStatistics) RemoveStream(name string) {
	stats.lock.Lock()
	delete(stats.internal, name)
	delete(stats.streams, name)
	stats.lock.Unlock()
}

// GetStreamStatistics fetches the statistics for a stream.
// The returned object is a copy does not need to be handled with care.
func (stats *realStatistics) GetStreamStatistics(name string) *StreamStatistics {
	stats.lock.RLock()
	stream := *stats.streams[name]
	stats.lock.RUnlock()
	return &stream
}

// GetAllStreamStatistics fetches the statistics for all streams.
// The returned object is a copy does not need to be handled with care.
func (stats *realStatistics) GetAllStreamStatistics() map[string]*StreamStatistics {
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
func (stats *realStatistics) GetGlobalStatistics() *StreamStatistics {
	stats.lock.RLock()
	global := *stats.global
	stats.lock.RUnlock()
	return &global
}

type dummyStatistics struct {
}

// NewDummyStatistics creates a placeholder statistics container.
// It doesn't actually do anything and is just used for debugging.
func NewDummyStatistics() Statistics {
	return &dummyStatistics{}
}

func (stats *dummyStatistics) Start() {
}

func (stats *dummyStatistics) Stop() {
}

func (stats *dummyStatistics) RegisterStream(name string) Collector {
	return &dummyCollector{}
}

func (stats *dummyStatistics) RemoveStream(name string) {
}

func (stats *dummyStatistics) GetStreamStatistics(name string) *StreamStatistics {
	return &StreamStatistics{}
}

func (stats *dummyStatistics) GetAllStreamStatistics() map[string]*StreamStatistics {
	return make(map[string]*StreamStatistics)
}

func (stats *dummyStatistics) GetGlobalStatistics() *StreamStatistics {
	return &StreamStatistics{}
}

type dummyCollector struct {
}

func (stats *dummyCollector) ConnectionAdded() {
}

func (stats *dummyCollector) ConnectionRemoved() {
}

func (stats *dummyCollector) PacketReceived() {
}

func (stats *dummyCollector) PacketSent() {
}

func (stats *dummyCollector) PacketDropped() {
}

func (stats *dummyCollector) SourceConnected() {
}

func (stats *dummyCollector) SourceDisconnected() {
}

func (stats *dummyCollector) IsUpstreamConnected() bool {
	return false
}
