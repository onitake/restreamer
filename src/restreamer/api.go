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
	"net/http"
	"encoding/json"
)

// healthApi encapsulates a system status object and
// provides an HTTP/JSON handler for reporting system health.
type healthApi struct {
	stats Statistics
}

// NewHealthApi creates a new health API object,
// serving data from a system Statistics object.
func NewHealthApi(stats Statistics) http.Handler {
	return &healthApi{
		stats: stats,
	}
}

// ServeHTTP is the http handler method.
// It sends back information about system health.
func (api *healthApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	global := api.stats.GetGlobalStatistics()
	var stats struct {
		Status string `json:"status"`
		Viewer int `json:"viewer"`
		Limit int `json:"limit"`
		Bandwidth int `json:"bandwidth"`
	}
	if global.Connections < global.MaxConnections {
		stats.Status = "ok"
	} else {
		stats.Status = "full"
	}
	stats.Viewer = int(global.Connections)
	stats.Limit = int(global.MaxConnections)
	stats.Bandwidth = int(global.BytesPerSecondSent * 8 / 1024) // kbit/s
	
	writer.Header().Add("Content-Type", "application/json")
	response, err := json.Marshal(&stats)
	if err == nil {
		writer.WriteHeader(http.StatusOK);
		writer.Write(response)
	} else {
		writer.WriteHeader(http.StatusInternalServerError);
		writer.Write([]byte("500 internal server error"))
		log.Print(err)
	}
}


// statisticsApi encapsulates a system status object and
// provides an HTTP/JSON handler for reporting total system statistics.
type statisticsApi struct {
	stats Statistics
}

// NewStatisticsApi creates a new statistics API object,
// serving data from a system Statistics object.
func NewStatisticsApi(stats Statistics) http.Handler {
	return &statisticsApi{
		stats: stats,
	}
}

// ServeHTTP is the http handler method.
// It sends back information about system health.
func (api *statisticsApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	global := api.stats.GetGlobalStatistics()
	var stats struct {
		Status string `json:"status"`
		Connections int `json:"connections"`
		MaxConnections int `json:"max_connections"`
		TotalPacketsReceived uint64 `json:"total_packets_received"`
		TotalPacketsSent uint64 `json:"total_packets_sent"`
		TotalPacketsDropped uint64 `json:"total_packets_dropped"`
		TotalBytesReceived uint64 `json:"total_bytes_received"`
		TotalBytesSent uint64 `json:"total_bytes_sent"`
		TotalBytesDropped uint64 `json:"total_bytes_dropped"`
		PacketsPerSecondReceived uint64 `json:"packets_per_second_received"`
		PacketsPerSecondSent uint64 `json:"packets_per_second_sent"`
		PacketsPerSecondDropped uint64 `json:"packets_per_second_dropped"`
		BytesPerSecondReceived uint64 `json:"bytes_per_second_received"`
		BytesPerSecondSent uint64 `json:"bytes_per_second_sent"`
		BytesPerSecondDropped uint64 `json:"bytes_per_second_dropped"`
	}
	if global.Connections < global.MaxConnections {
		stats.Status = "ok"
	} else {
		stats.Status = "full"
	}
	stats.Connections = int(global.Connections)
	stats.MaxConnections = int(global.MaxConnections)
	stats.TotalPacketsReceived = global.TotalPacketsReceived
	stats.TotalPacketsSent = global.TotalPacketsSent
	stats.TotalPacketsDropped = global.TotalPacketsDropped
	stats.TotalBytesReceived = global.TotalBytesReceived
	stats.TotalBytesSent = global.TotalBytesSent
	stats.TotalBytesDropped = global.TotalBytesDropped
	stats.PacketsPerSecondReceived = global.PacketsPerSecondReceived
	stats.PacketsPerSecondSent = global.PacketsPerSecondSent
	stats.PacketsPerSecondDropped = global.PacketsPerSecondDropped
	stats.BytesPerSecondReceived = global.BytesPerSecondReceived
	stats.BytesPerSecondSent = global.BytesPerSecondSent
	stats.BytesPerSecondDropped = global.BytesPerSecondDropped
	
	writer.Header().Add("Content-Type", "application/json")
	response, err := json.Marshal(&stats)
	if err == nil {
		writer.WriteHeader(http.StatusOK);
		writer.Write(response)
	} else {
		writer.WriteHeader(http.StatusInternalServerError);
		writer.Write([]byte("500 internal server error"))
		log.Print(err)
	}
}

// StreamStatApi provides an API for checking stream availability.
// The HTTP handler returns status code 200 if a stream is connected
// and 404 if not.
type streamStateApi struct {
	client *Client
}

// NewStreamStateApi creates a new stream status API object,
// serving the "connected" status of a stream connection.
func NewStreamStateApi(client *Client) http.Handler {
	return &streamStateApi{
		client: client,
	}
}

// ServeHTTP is the http handler method.
// It sends back "200 ok" if the stream is connected and "404 not found" if not,
// along with the corresponding HTTP status code.
func (stat *streamStateApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/plain")
	if stat.client.Connected() {
		writer.WriteHeader(http.StatusOK);
		writer.Write([]byte("200 ok"))
	} else {
		writer.WriteHeader(http.StatusNotFound);
		writer.Write([]byte("404 not found"))
	}
}

 
