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
	"time"
	"net/http"
	"encoding/json"
)

// healthApi encapsulates a system status object and
// provides an HTTP/JSON handler for reporting system health.
type healthApi struct {
	stats *Statistics
}

// NewHealthApi creates a new health API object,
// serving data from a system Statistics object.
func (stats *Statistics) NewHealthApi() http.Handler {
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
	stats.Bandwidth = int(global.BytesPerSecondSent * 8 / 1024)
	
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

// statsApi encapsulates a system status object and
// provides an HTTP/JSON handler for reporting system statistics.
type statsApi struct {
	stats *Statistics
}

// NewStatsApi creates a new health API object,
// serving data from a system Statistics object.
func (stats *Statistics) NewStatsApi() http.Handler {
	return &statsApi{
		stats: stats,
	}
}

// ServeHTTP is the http handler method.
// It sends back system statistics.
func (api *statsApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	response, err := json.Marshal(map[string]interface{}{
		"lastUpdate": time.Now().Unix(),
		"total": map[string]interface{}{
			"counter": 0,
			"free": 1000,
		},
		"servers": []interface{}{
			map[string]interface{}{
				"counter": 0,
				"name": "streaming-test.local",
				"free": 1000,
			},
		},
	})
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
type streamStatApi struct {
	client *Client
}

// NewStreamStatApi creates a new stream status API object,
// serving the "connected" status of a stream connection.
func (*Statistics) NewStreamStatApi(client *Client) http.Handler {
	return &streamStatApi{
		client: client,
	}
}

// ServeHTTP is the http handler method.
// It sends back "200 ok" if the stream is connected and "404 not found" if not,
// along with the corresponding HTTP status code.
func (stat *streamStatApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/plain")
	if stat.client.Connected() {
		writer.WriteHeader(http.StatusOK);
		writer.Write([]byte("200 ok"))
	} else {
		writer.WriteHeader(http.StatusNotFound);
		writer.Write([]byte("404 not found"))
	}
}

 
