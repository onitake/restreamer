/* Copyright (c) 2016-2018 Gregor Riepl
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
	"encoding/json"
	"github.com/onitake/restreamer/auth"
	"github.com/onitake/restreamer/metrics"
	"net/http"
)

// connectChecker represents a type that can report its "connected" status.
type connectChecker interface {
	Connected() bool
}

// healthApi encapsulates a system status object and
// provides an HTTP/JSON handler for reporting system health.
type healthApi struct {
	stats metrics.Statistics
	// auth is an authentication verifier for client requests
	auth auth.Authenticator
}

// NewHealthApi creates a new health API object,
// serving data from a system Statistics object.
func NewHealthApi(stats metrics.Statistics, auth auth.Authenticator) http.Handler {
	return &healthApi{
		stats: stats,
		auth:  auth,
	}
}

// ServeHTTP is the http handler method.
// It sends back information about system health.
func (api *healthApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// set the content type for all responses
	writer.Header().Add("Content-Type", "application/json")

	// fail-fast: verify that this user can access this resource first
	if !auth.HandleHttpAuthentication(api.auth, request, writer) {
		return
	}

	global := api.stats.GetGlobalStatistics()
	var stats struct {
		Status    string `json:"status"`
		Viewer    int    `json:"viewer"`
		Limit     int    `json:"limit"`
		Max       int    `json:"max"`
		Bandwidth int    `json:"bandwidth"`
	}
	// report for both hard and soft, respecting disabled limits
	if global.MaxConnections != 0 && global.Connections >= global.MaxConnections {
		stats.Status = "full"
	} else if global.FullConnections != 0 && global.Connections >= global.FullConnections {
		stats.Status = "full"
	} else {
		stats.Status = "ok"
	}
	stats.Viewer = int(global.Connections)
	stats.Limit = int(global.FullConnections)
	stats.Max = int(global.MaxConnections)
	stats.Bandwidth = int(global.BytesPerSecondSent * 8 / 1024) // kbit/s

	response, err := json.Marshal(&stats)
	if err == nil {
		writer.WriteHeader(http.StatusOK)
		writer.Write(response)
	} else {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		logger.Logkv(
			"event", eventApiError,
			"error", errorApiJsonEncode,
			"message", err.Error(),
		)
	}
}

// statisticsApi encapsulates a system status object and
// provides an HTTP/JSON handler for reporting total system statistics.
type statisticsApi struct {
	stats metrics.Statistics
	// auth is an authentication verifier for client requests
	auth auth.Authenticator
}

// NewStatisticsApi creates a new statistics API object,
// serving data from a system Statistics object.
func NewStatisticsApi(stats metrics.Statistics, auth auth.Authenticator) http.Handler {
	return &statisticsApi{
		stats: stats,
		auth:  auth,
	}
}

// ServeHTTP is the http handler method.
// It sends back information about system health.
func (api *statisticsApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// set the content type for all responses
	writer.Header().Add("Content-Type", "application/json")

	// fail-fast: verify that this user can access this resource first
	if !auth.HandleHttpAuthentication(api.auth, request, writer) {
		return
	}

	global := api.stats.GetGlobalStatistics()
	var stats struct {
		Status                   string `json:"status"`
		Connections              int    `json:"connections"`
		MaxConnections           int    `json:"max_connections"`
		FullConnections          int    `json:"full_connections"`
		TotalPacketsReceived     uint64 `json:"total_packets_received"`
		TotalPacketsSent         uint64 `json:"total_packets_sent"`
		TotalPacketsDropped      uint64 `json:"total_packets_dropped"`
		TotalBytesReceived       uint64 `json:"total_bytes_received"`
		TotalBytesSent           uint64 `json:"total_bytes_sent"`
		TotalBytesDropped        uint64 `json:"total_bytes_dropped"`
		TotalStreamTime          int64  `json:"total_stream_time_ns"`
		PacketsPerSecondReceived uint64 `json:"packets_per_second_received"`
		PacketsPerSecondSent     uint64 `json:"packets_per_second_sent"`
		PacketsPerSecondDropped  uint64 `json:"packets_per_second_dropped"`
		BytesPerSecondReceived   uint64 `json:"bytes_per_second_received"`
		BytesPerSecondSent       uint64 `json:"bytes_per_second_sent"`
		BytesPerSecondDropped    uint64 `json:"bytes_per_second_dropped"`
	}
	// report for both hard and soft, respecting disabled limits
	if global.MaxConnections != 0 && global.Connections >= global.MaxConnections {
		stats.Status = "overload"
	} else if global.FullConnections != 0 && global.Connections >= global.FullConnections {
		stats.Status = "full"
	} else {
		stats.Status = "ok"
	}
	stats.Connections = int(global.Connections)
	stats.MaxConnections = int(global.MaxConnections)
	stats.FullConnections = int(global.FullConnections)
	stats.TotalPacketsReceived = global.TotalPacketsReceived
	stats.TotalPacketsSent = global.TotalPacketsSent
	stats.TotalPacketsDropped = global.TotalPacketsDropped
	stats.TotalBytesReceived = global.TotalBytesReceived
	stats.TotalBytesSent = global.TotalBytesSent
	stats.TotalBytesDropped = global.TotalBytesDropped
	stats.TotalStreamTime = global.TotalStreamTime
	stats.PacketsPerSecondReceived = global.PacketsPerSecondReceived
	stats.PacketsPerSecondSent = global.PacketsPerSecondSent
	stats.PacketsPerSecondDropped = global.PacketsPerSecondDropped
	stats.BytesPerSecondReceived = global.BytesPerSecondReceived
	stats.BytesPerSecondSent = global.BytesPerSecondSent
	stats.BytesPerSecondDropped = global.BytesPerSecondDropped

	response, err := json.Marshal(&stats)
	if err == nil {
		writer.WriteHeader(http.StatusOK)
		writer.Write(response)
	} else {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		logger.Logkv(
			"event", eventApiError,
			"error", errorApiJsonEncode,
			"message", err.Error(),
		)
	}
}

// streamStatApi provides an API for checking stream availability.
// The HTTP handler returns status code 200 if a stream is connected
// and 404 if not.
type streamStateApi struct {
	client connectChecker
	// auth is an authentication verifier for client requests
	auth auth.Authenticator
}

// NewStreamStateApi creates a new stream status API object,
// serving the "connected" status of a stream connection.
func NewStreamStateApi(client connectChecker, auth auth.Authenticator) http.Handler {
	return &streamStateApi{
		client: client,
		auth:   auth,
	}
}

// ServeHTTP is the http handler method.
// It sends back "200 ok" if the stream is connected and "404 not found" if not,
// along with the corresponding HTTP status code.
func (api *streamStateApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// set the content type for all responses
	writer.Header().Add("Content-Type", "text/plain")

	// fail-fast: verify that this user can access this resource first
	if !auth.HandleHttpAuthentication(api.auth, request, writer) {
		return
	}

	if api.client.Connected() {
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("200 ok"))
	} else {
		writer.WriteHeader(http.StatusNotFound)
		writer.Write([]byte("404 not found"))
	}
}

// inhibitor represents a type that can prevent or allow new connections.
type inhibitor interface {
	SetInhibit(inhibit bool)
}

// streamControlApi allows manipulation of a stream's state.
// If this API is enabled for a stream, requests to start and stop it externally
// can be sent. Useful for testing or as an emergency kill switch.
type streamControlApi struct {
	inhibit inhibitor
	// auth is an authentication verifier for client requests
	auth auth.Authenticator
}

// NewStreamStateApi creates a new stream status API object,
// serving the "connected" status of a stream connection.
func NewStreamControlApi(inhibit inhibitor, auth auth.Authenticator) http.Handler {
	return &streamControlApi{
		inhibit: inhibit,
		auth:    auth,
	}
}

// ServeHTTP is the http handler method.
// It parses the query string and prohibits or allows new connections depending
// on the existence of the "offline" or "online" parameter.
// When the "offline" parameter is present, all existing downstream connections
// are closed immediately. If both are present, the query is treated like
// if there was only "offline".
func (api *streamControlApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// set the content type for all responses
	writer.Header().Add("Content-Type", "text/plain")

	// fail-fast: verify that this user can access this resource first
	if !auth.HandleHttpAuthentication(api.auth, request, writer) {
		return
	}

	query := request.URL.Query()
	if len(query["offline"]) > 0 {
		api.inhibit.SetInhibit(true)
		writer.WriteHeader(http.StatusAccepted)
		writer.Write([]byte("202 accepted"))
	} else if len(query["online"]) > 0 {
		api.inhibit.SetInhibit(false)
		writer.WriteHeader(http.StatusAccepted)
		writer.Write([]byte("202 accepted"))
	} else {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte("400 bad request"))
	}
}

// prometheusApi implements a handler for scraping Prometheus metrics.
type prometheusApi struct {
	// auth is an authentication verifier for client requests
	auth auth.Authenticator
	// handler is the delegate HTTP handler
	handler http.Handler
}

// NewPrometheusApi creates a new Prometheus metrics API object,
// serving metrics to a Prometheus instance.
func NewPrometheusApi(auth auth.Authenticator) http.Handler {
	return &prometheusApi{
		auth:    auth,
		handler: metrics.PromHandler(),
	}
}

// ServeHTTP is the http handler method.
func (api *prometheusApi) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// fail-fast: verify that this user can access this resource first
	if !auth.HandleHttpAuthentication(api.auth, request, writer) {
		return
	}

	// authentication successful, forward the request to the promhttp handler
	api.handler.ServeHTTP(writer, request)
}
