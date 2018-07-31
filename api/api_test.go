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
	//"encoding/hex"
	"bytes"
	"encoding/json"
	"github.com/onitake/restreamer/configuration"
	"github.com/onitake/restreamer/protocol"
	"net/http"
	"net/url"
	"testing"
)

type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

type mockWriter struct {
	bytes.Buffer
	header http.Header
	log    Logger
}

func newMockWriter(logger Logger) *mockWriter {
	return &mockWriter{
		header: make(http.Header),
		log:    logger,
	}
}
func (writer *mockWriter) Header() http.Header {
	return writer.header
}
func (writer *mockWriter) Write(data []byte) (int, error) {
	//writer.log.Logf("Write data:\n%s", hex.Dump(data))
	return writer.Buffer.Write(data)
}
func (writer *mockWriter) WriteHeader(status int) {
	//writer.log.Logf("Write header, status code %d:", status)
	//writer.log.Log(writer.header)
}

type mockStatistics struct {
	Streams map[string]*StreamStatistics
	Global  StreamStatistics
}

func (*mockStatistics) Start() {}
func (*mockStatistics) Stop()  {}
func (*mockStatistics) RegisterStream(name string) Collector {
	return nil
}
func (*mockStatistics) RemoveStream(name string) {}
func (stats *mockStatistics) GetStreamStatistics(name string) *StreamStatistics {
	return stats.Streams[name]
}
func (stats *mockStatistics) GetAllStreamStatistics() map[string]*StreamStatistics {
	return stats.Streams
}
func (stats *mockStatistics) GetGlobalStatistics() *StreamStatistics {
	return &stats.Global
}

func testStatisticsConnections(t *testing.T, connections, full, max int64, status string) {
	stats := &mockStatistics{
		Global: StreamStatistics{
			Connections:     connections,
			MaxConnections:  max,
			FullConnections: full,
		},
	}
	api := &statisticsApi{
		stats: stats,
		auth:  protocol.NewAuthenticator(configuration.Authentication{}, nil),
	}
	writer := newMockWriter(t)
	testurl, _ := url.Parse("http://localhost/statistics")
	api.ServeHTTP(writer, &http.Request{Header: make(http.Header), URL: testurl})
	var decoded map[string]interface{}
	err := json.Unmarshal(writer.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err.Error())
	}
	retstatus, ok := decoded["status"].(string)
	if !ok {
		t.Fatalf("No status field or incorrect type returned")
	}
	if retstatus != status {
		t.Errorf("Invalid status returned: expected %s, got %s", status, retstatus)
	}
}

func testHealthConnections(t *testing.T, connections, full, max int64, status string) {
	stats := &mockStatistics{
		Global: StreamStatistics{
			Connections:     connections,
			MaxConnections:  max,
			FullConnections: full,
		},
	}
	api := &healthApi{
		stats: stats,
		auth:  protocol.NewAuthenticator(configuration.Authentication{}, nil),
	}
	writer := newMockWriter(t)
	testurl, _ := url.Parse("http://localhost/health")
	api.ServeHTTP(writer, &http.Request{Header: make(http.Header), URL: testurl})
	var decoded map[string]interface{}
	err := json.Unmarshal(writer.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err.Error())
	}
	retstatus, ok := decoded["status"].(string)
	if !ok {
		t.Fatalf("No status field or incorrect type returned")
	}
	if retstatus != status {
		t.Errorf("Invalid status returned: expected %s, got %s", status, retstatus)
	}
}

func TestStatisticsApi(t *testing.T) {
	testStatisticsConnections(t, 0, 0, 0, "ok")
	testStatisticsConnections(t, 1, 0, 0, "ok")
	testStatisticsConnections(t, 1, 1, 2, "full")
	testStatisticsConnections(t, 1, 1, 0, "full")
	testStatisticsConnections(t, 1, 0, 2, "ok")
	testStatisticsConnections(t, 2, 1, 2, "overload")
	testStatisticsConnections(t, 2, 1, 0, "full")
	testStatisticsConnections(t, 2, 0, 2, "overload")
}

func TestHealthApi(t *testing.T) {
	testHealthConnections(t, 0, 0, 0, "ok")
	testHealthConnections(t, 1, 0, 0, "ok")
	testHealthConnections(t, 1, 1, 2, "full")
	testHealthConnections(t, 1, 1, 0, "full")
	testHealthConnections(t, 1, 0, 2, "ok")
	testHealthConnections(t, 2, 1, 2, "full")
	testHealthConnections(t, 2, 1, 0, "full")
	testHealthConnections(t, 2, 0, 2, "full")
}
