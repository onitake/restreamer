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
	"errors"
	"sort"
)

var (
	ErrMetricDoesNotExist = errors.New("Metric does not exist")
)

type requestType int

const (
	updateRequest requestType = iota
	fetchRequest
	stopRequest
)

// Metric represents a single metric and includes
// - a name
// - the metric data
// - optionally a set of tags.
//
// Tags are used to separate metrics with the same name into different buckets.
// They can be used to store host names, namespaces, modules, etc.
//
// Note: The name and all tags and their values can use all Unicode characters
// representable by a rune, EXCEPT for the null character (\0).
// The null character is used to construct a combined metric key for efficient lookup.
type Metric struct {
	// Name is the name of the metric.
	// May not contain null (\0) characters.
	Name string
	// Tags is a dictionary containing tags with values. It can be empty.
	// May not contain null (\0) characters in tags or tag values.
	Tags map[string]string
	// Value is the value stored in the metric
	Value Datum
}

// MakeKey generates a unique key for storing this metric.
//
// The key is constructed as follows:
// Key = Name
// TagKeys = SORT_LEXICAL(KEYS(Tags))
// FOR EACH TagKey IN TagKeys:
//   Key = CONCAT(Key, '\0', TagKey, '\0', VALUE_FOR_KEY(Tags, TagKey))
func (m *Metric) MakeKey() string {
	key := m.Name
	tagkeys := make([]string, 0, len(m.Tags))
	for k, _ := range m.Tags {
		tagkeys = append(tagkeys, k)
	}
	sort.Strings(tagkeys)
	for _, k := range tagkeys {
		key += "\u0000"
		key += k
		key += "\u0000"
		key += m.Tags[k]
	}
	return key
}

// MetricResponse contains a reponse to a single metric update or fetch request.
type MetricResponse struct {
	// Error is nil on a successul update or non-nil if the update or fetch failed.
	Error error
	// Metric contains the new value of the metric.
	Metric Metric
}

// metricsUpdate represents a single update, fetch or stop request.
// Multiple metrics can be updated in a single request.
type metricsUpdate struct {
	// Type is the type of request
	Type requestType
	// Metrics is the list of metrics to update
	Metrics []Metric
	// Return is a return channel to send responses or errors back to the requester.
	// Can be nil if no response is required.
	Return chan<- []MetricResponse
}

type MetricsCollector interface {
	Stop()
	Update(metrics []Metric, ret chan<- []MetricResponse)
	Fetch(metrics []Metric, ret chan<- []MetricResponse)
}

// DummyMetricsCollector is a metrics collector that doesn't collect anything
// and simply returns errors when trying to fetch data
type DummyMetricsCollector struct{}

func (c *DummyMetricsCollector) Stop() {
	// nothing
}

// Update updates one or more metrics.
// If a metric doesn't exist, it will be created.
// It it exists and has a different type, an error is generated and no update will occur.
// When passing multiple metrics to update, one failure will not prevent other metrics from being updated.
func (c *DummyMetricsCollector) Update(metrics []Metric, ret chan<- []MetricResponse) {
	if ret != nil {
		empty := make([]MetricResponse, len(metrics))
		ret <- empty
	}
}

// Fetches one or more metrics.
// Any values passed with the request are ignored.
// If a metric doesn't exist, an error will be generated.
func (c *DummyMetricsCollector) Fetch(metrics []Metric, ret chan<- []MetricResponse) {
	if ret != nil {
		empty := make([]MetricResponse, len(metrics))
		for _, m := range empty {
			m.Error = ErrMetricDoesNotExist
		}
		ret <- empty
	}
}

// realMetricsCollector is a generic metrics collector for any kind of system metrics.
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
type realMetricsCollector struct {
	// queue is the processing queue for updates
	queue chan *metricsUpdate
	// metrics contains all the collected metrics.
	//
	// It is keyed by a unique combination of the Name and all Tags and their
	// Values associated with each metric.
	// See Metric.MakeKey() for a description of the algorithm.
	metrics map[string]*Metric
}

// NewMetricsCollector creates a new metrics collector and starts its background processor.
func NewMetricsCollector() MetricsCollector {
	c := &realMetricsCollector{
		queue:   make(chan *metricsUpdate),
		metrics: make(map[string]*Metric),
	}
	c.start()
	return c
}

// start starts the collection loop
func (c *realMetricsCollector) start() {
	go c.loop()
}

func (c *realMetricsCollector) loop() {
	running := true
	for running {
		msg := <-c.queue
		switch msg.Type {
		case updateRequest:
			response := make([]MetricResponse, 0, len(msg.Metrics))
			for _, m := range msg.Metrics {
				pm := &m
				r := MetricResponse{}
				k := pm.MakeKey()
				if c.metrics[k] == nil {
					c.metrics[k] = pm
				} else {
					r.Error = c.metrics[k].Value.UpdateFrom(pm.Value)
				}
				r.Metric = *c.metrics[k]
				response = append(response, r)
			}
			if msg.Return != nil {
				msg.Return <- response
			}
		case fetchRequest:
			response := make([]MetricResponse, 0, len(msg.Metrics))
			for _, m := range msg.Metrics {
				pm := &m
				r := MetricResponse{}
				k := pm.MakeKey()
				if c.metrics[k] == nil {
					r.Error = ErrMetricDoesNotExist
				} else {
					r.Metric = *c.metrics[k]
				}
				response = append(response, r)
			}
			if msg.Return != nil {
				msg.Return <- response
			}
		case stopRequest:
			running = false
		default:
			panic("Invalid metric type")
		}
	}
}

// Stop stops the collection loop
func (c *realMetricsCollector) Stop() {
	c.queue <- &metricsUpdate{stopRequest, nil, nil}
}

// Update updates one or more metrics.
// If a metric doesn't exist, it will be created.
// It it exists and has a different type, an error is generated and no update will occur.
// When passing multiple metrics to update, one failure will not prevent other metrics from being updated.
func (c *realMetricsCollector) Update(metrics []Metric, ret chan<- []MetricResponse) {
	c.queue <- &metricsUpdate{updateRequest, metrics, ret}
}

// Fetches one or more metrics.
// Any values passed with the request are ignored.
// If a metric doesn't exist, an error will be generated.
func (c *realMetricsCollector) Fetch(metrics []Metric, ret chan<- []MetricResponse) {
	c.queue <- &metricsUpdate{fetchRequest, metrics, ret}
}
