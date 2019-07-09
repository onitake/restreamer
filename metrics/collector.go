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

type requestType int

const (
	updateRequest requestType = iota
	fetchRequest
	stopRequest
)

// metricUpdate represents a single update, fetch or stop request
type metricUpdate struct {
	// typ is the type of request
	typ requestType
	// namespace is the directory where the metric is stored
	namespace string
	// metric is the name of the metric to update
	metric string
	// datum contains a reference to the datum to send or receive
	// its type must match the metric, if it already exists
	// if it doesn't, a new metric is created
	datum Datum
}

// MetricsCollector is a generic metrics collector for any kind of system metrics.
//
// It uses a channel to transport updates to a central data processor.
//
// Supported metric types are:
// - Gauge: a value that is overwritten on every update
// - Counter: a value that is counted up or down on every update
//
// Supported data types are:
// - int64: signed 64-bit integer
// - float64: signed 64-bit floating-point
// - bool: boolean (only for gauges)
// - string: string (only for gauges)
type MetricsCollector struct {
	// queue is the processing queue for updates
	queue chan *metricUpdate
	// metrics contains all the collected metrics,
	// sorted by namespace and metric name
	metrics map[string]map[string]Datum
}

// NewMetricsCollector creates a new metrics collector and starts its background processor.
func NewMetricsCollector() *MetricsCollector {
	c := &MetricsCollector{
		queue:   make(chan *metricUpdate),
		metrics: make(map[string]map[string]Datum),
	}
	c.start()
	return c
}

// start starts the collection loop
func (c *MetricsCollector) start() {
	go c.loop()
}

func (c *MetricsCollector) loop() {
	running := true
	for running {
		msg := <-c.queue
		switch msg.typ {
		case updateRequest:
			if c.metrics[msg.namespace] == nil {
				c.metrics[msg.namespace] = make(map[string]Datum)
			}
			if c.metrics[msg.namespace][msg.metric] == nil {
				c.metrics[msg.namespace][msg.metric] = msg.datum
			} else {
				// TODO process the error
				/*err = */
				c.metrics[msg.namespace][msg.metric].UpdateFrom(msg.datum)
			}
		case fetchRequest:
			if c.metrics[msg.namespace] == nil || c.metrics[msg.namespace][msg.metric] == nil {
				// TODO return an error
			} else {
				// TODO return a copy of c.metrics[msg.namespace][msg.metric]
			}
		case stopRequest:
			running = false
		default:
			panic("Invalid metric type")
		}
	}
}

// Stop stops the collection loop
func (c *MetricsCollector) Stop() {
	c.queue <- &metricUpdate{stopRequest, "", "", nil}
}

// UpdateIntGauge updates an int64 gauge.
func (c *MetricsCollector) UpdateIntGauge(namespace, metric string, datum int64) {
	pdatum := int64Gauge(datum)
	c.queue <- &metricUpdate{updateRequest, namespace, metric, &pdatum}
}

// UpdateFloatGauge updates a float64 gauge.
func (c *MetricsCollector) UpdateFloatGauge(namespace, metric string, datum float64) {
	pdatum := float64Gauge(datum)
	c.queue <- &metricUpdate{updateRequest, namespace, metric, &pdatum}
}

// UpdateBoolGauge updates a boolean gauge.
func (c *MetricsCollector) UpdateBoolGauge(namespace, metric string, datum bool) {
	pdatum := boolGauge(datum)
	c.queue <- &metricUpdate{updateRequest, namespace, metric, &pdatum}
}

// UpdateStringGauge updates a string gauge.
func (c *MetricsCollector) UpdateStringGauge(namespace, metric string, datum string) {
	pdatum := stringGauge(datum)
	c.queue <- &metricUpdate{updateRequest, namespace, metric, &pdatum}
}

// AddIntCounter adds datum to an int64 counter.
func (c *MetricsCollector) AddIntCounter(namespace, metric string, datum int64) {
	pdatum := int64Counter(datum)
	c.queue <- &metricUpdate{updateRequest, namespace, metric, &pdatum}
}

// AddFloatCounter adds datum to a float64 counter.
func (c *MetricsCollector) AddFloatCounter(namespace, metric string, datum float64) {
	pdatum := float64Counter(datum)
	c.queue <- &metricUpdate{updateRequest, namespace, metric, &pdatum}
}
