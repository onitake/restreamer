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

package api

import (
	"testing"
	"time"
)

/*
func (stats *DummyStatistics) Start() {
func (stats *DummyStatistics) Stop() {
func (stats *DummyStatistics) RegisterStream(name string) Collector {
func (stats *DummyStatistics) RemoveStream(name string) {
func (stats *DummyStatistics) GetStreamStatistics(name string) *StreamStatistics {
func (stats *DummyStatistics) GetAllStreamStatistics() map[string]*StreamStatistics {
func (stats *DummyStatistics) GetGlobalStatistics() *StreamStatistics {

func (stats *DummyCollector) ConnectionAdded() {
func (stats *DummyCollector) ConnectionRemoved() {
func (stats *DummyCollector) PacketReceived() {
func (stats *DummyCollector) PacketSent() {
func (stats *DummyCollector) PacketDropped() {
func (stats *DummyCollector) SourceConnected() {
func (stats *DummyCollector) SourceDisconnected() {
func (stats *DummyCollector) IsUpstreamConnected() bool {
func (stats *DummyCollector) StreamDuration(duration time.Duration) {

Connections              int64
MaxConnections           int64
FullConnections          int64
TotalPacketsReceived     uint64
TotalPacketsSent         uint64
TotalPacketsDropped      uint64
TotalBytesReceived       uint64
TotalBytesSent           uint64
TotalBytesDropped        uint64
TotalStreamTime          int64
PacketsPerSecondReceived uint64
PacketsPerSecondSent     uint64
PacketsPerSecondDropped  uint64
BytesPerSecondReceived   uint64
BytesPerSecondSent       uint64
BytesPerSecondDropped    uint64
Connected                bool
*/

func testStatistics01(t *testing.T, s Statistics) {
	s.Start()
	s.Stop()
}

func testStatistics02(t *testing.T, s Statistics) {
	s.RegisterStream("t02")
	s.RemoveStream("t02")
}

func testStatistics03(t *testing.T, s Statistics) {
	s.RegisterStream("t03")
	s.GetStreamStatistics("t03")
	s.RemoveStream("t03")
}

func testStatistics04(t *testing.T, s Statistics, max, full int64) {
	r := s.GetGlobalStatistics()
	if r.MaxConnections != max {
		t.Errorf("t04: Max connection value (=%v) not matched (=%v)", r.MaxConnections, max)
	}
	if r.FullConnections != full {
		t.Errorf("t04: Full connection value (=%v) not matched (=%v)", r.FullConnections, full)
	}
}

func testStatistics05(t *testing.T, s Statistics) {
	c := s.RegisterStream("t05")
	s.Start()
	<-time.After(1 * time.Second)
	c.ConnectionAdded()
	c.PacketReceived()
	<-time.After(1 * time.Second)
	r := s.GetStreamStatistics("t05")
	s.Stop()
	t.Logf("t05: %v", r)
	s.RemoveStream("t05")
	if r.Connections != 1 {
		t.Errorf("t05: Connected value (=%v) not matched (=%v)", r.Connections, 1)
	}
}

func TestDummyStatistics(t *testing.T) {
	testStatistics01(t, &DummyStatistics{})
	testStatistics02(t, &DummyStatistics{})
	testStatistics03(t, &DummyStatistics{})
}

func TestRealStatistics(t *testing.T) {
	testStatistics01(t, NewStatistics(0, 0))
	testStatistics02(t, NewStatistics(0, 0))
	testStatistics03(t, NewStatistics(0, 0))
	testStatistics04(t, NewStatistics(10, 20), 10, 20)
	testStatistics05(t, NewStatistics(0, 0))
}
