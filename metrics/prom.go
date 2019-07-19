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

package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	defaultRegistry = prometheus.NewRegistry()
	// DefaultRegisterer is a prometheus client registry that contains no
	// default metrics. See prometheus.Registry for more information.
	DefaultRegisterer prometheus.Registerer = defaultRegistry
	// DefaultGatherer points to the same registry as DefaultRegisterer.
	// See prometheus.Registry for more information.
	DefaultGatherer prometheus.Gatherer = defaultRegistry
)

// promErrorLogger is an internal error logger that prints to the kvl log.
type promErrorLogger struct{}

func (*promErrorLogger) Println(v ...interface{}) {
	logger.Logkv(
		"event", eventMetricsError,
		"error", errorMetricsPrometheus,
		"message", fmt.Sprintln(v...),
	)
}

// PromHandler creates a prometheus HTTP handler that wraps DefaultGatherer
// and logs to the standard kvl logger.
func PromHandler() http.Handler {
	return promhttp.HandlerFor(DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog:      &promErrorLogger{},
		ErrorHandling: promhttp.ContinueOnError,
	})
}

// MustRegister registers the provided Collectors with the DefaultRegisterer
// and panics if any error occurs.
//
// MustRegister is a shortcut for DefaultRegisterer.MustRegister(cs...). See
// there for more details.
func MustRegister(cs ...prometheus.Collector) {
	DefaultRegisterer.MustRegister(cs...)
}

// Register registers the provided Collector with the DefaultRegisterer.
//
// Register is a shortcut for DefaultRegisterer.Register(c). See there for more
// details.
func Register(c prometheus.Collector) error {
	return DefaultRegisterer.Register(c)
}
