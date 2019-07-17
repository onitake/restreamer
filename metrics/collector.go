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

// makeKey generates a unique key for storing this metric.
//
// The key is constructed as follows:
// Key = Name
// TagKeys = SORT_LEXICAL(KEYS(Tags))
// FOR EACH TagKey IN TagKeys:
//   Key = CONCAT(Key, '\0', TagKey, '\0', VALUE_FOR_KEY(Tags, TagKey))
func (m *Metric) makeKey() string {
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

// MetricsCollector is the generic interface for a metrics-collecting facility.
//
// Functions to stop collection, update a list of metrics and a fetching a
// filtered list of metrics are provided.
type MetricsCollector interface {
	// Stops metrics collection.
	// After calling this method, the collector should not be used any more.
	Stop()
	// Update updates one or more metrics.
	// If a metric doesn't exist, it will be created.
	// It it exists and has a different type, an error is generated and no update will occur.
	// When passing multiple metrics to update, one failure will not prevent other metrics from being updated.
	Update(metrics []Metric, ret chan<- []MetricResponse)
	// Fetch fetches one or more metrics.
	// Instead of a metric list, a filter must be provided.
	Fetch(include, exclude MetricFilter, ret chan<- []Metric)
}

// DummyMetricsCollector is a metrics collector that doesn't collect anything.
// It will still notify on the response channels, but with empty data.
type DummyMetricsCollector struct{}

func (c *DummyMetricsCollector) Stop() {
	// nothing
}

func (c *DummyMetricsCollector) Update(metrics []Metric, ret chan<- []MetricResponse) {
	if ret != nil {
		ret <- []MetricResponse{}
	}
}

func (c *DummyMetricsCollector) Fetch(include, exclude MetricFilter, ret chan<- []Metric) {
	if ret != nil {
		ret <- []Metric{}
	}
}

// metricsAction represents a single update, fetch or stop request.
// Multiple metrics can be updated in a single request.
type metricsAction interface {
	// act executes an action on the given collector.
	// The return value determines if the collector should continue or not
	// (used for the Stop action). false = stop.
	act(collector *realMetricsCollector) bool
}

type updateAction struct {
	// Metrics is the list of metrics to update
	metrics []Metric
	// Return is a return channel to send responses or errors back to the requester.
	// Can be nil if no response is required.
	ret chan<- []MetricResponse
}

func (a *updateAction) act(c *realMetricsCollector) bool {
	response := make([]MetricResponse, 0, len(a.metrics))
	for _, m := range a.metrics {
		pm := &m
		r := MetricResponse{}
		k := pm.makeKey()
		if c.metrics[k] == nil {
			c.metrics[k] = pm
		} else {
			r.Error = c.metrics[k].Value.UpdateFrom(pm.Value)
		}
		r.Metric = *c.metrics[k]
		response = append(response, r)
	}
	if a.ret != nil {
		a.ret <- response
	}
	return true;
}

type stopAction struct{}

func (a *stopAction) act(c *realMetricsCollector) bool {
	return false;
}

// MetricFilter defines which metrics need to match to be included in the result.
type MetricFilter struct {
	// Name refers to the metric name.
	// The empty string means all metrics
	Name string
	// Tags is a list of tag-values that must match.
	// If a value is the empty string, all metrics that include the tag match.
	Tags map[string]string
}

type fetchAction struct {
	// include defines which criteria need to be fulfilled by this fetch request.
	//
	// This is an inclusive list, anything that does not match the name or tags is excluded.
	// An empty name or tag list is treated as undefined, i.e. does not filter results.
	// A tag key with an empty value names a tag that must be set, but its value is not relevant.
	//
	// Inclusions are processed before exclusions.
	include MetricFilter
	// exclude defines which criteria must not be fulfilled by this fetch request.
	//
	// This is an exclusive list, anything that matches the name or tags is excluded.
	// An empty name or tag list is treated as undefined, i.e. does not filter results.
	// A tag key with an empty value names a tag that must not be set, independent of the value.
	exclude MetricFilter
	// Return is a return channel to send responses or errors back to the requester.
	ret chan<- []Metric
}

func (a *fetchAction) act(c *realMetricsCollector) bool {
	response := make([]Metric, 0)
	/*
	if a.include.name == nil {
		// start with all metric values
		for k, _ := range c.metrics {
			all = append(all, k)
		}
	} else {
		// start with a specific metric name
		all = append(all, c.nameBucket...)
	}
	for k, v := range a.include.tags {
		if v == "" {
			// filter by tag existence
			
		} else {
			// filter by tag and value
		}
		pm := &m
		r := MetricResponse{}
		k := pm.makeKey()
		if c.metrics[k] == nil {
			r.Error = ErrMetricDoesNotExist
		} else {
			r.Metric = *c.metrics[k]
		}
		response = append(response, r)
	}
	*/
	if a.ret != nil {
		a.ret <- response
	}
	return true;
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
	// queue is the processing queue for updates.
	queue chan metricsAction
	// metrics contains all the collected metrics.
	// It is keyed by a unique combination of the Name and all Tags and their
	// Values associated with each metric.
	// See Metric.makeKey() for a description of the algorithm.
	metrics map[string]*Metric
	// nameBucket maps metric names to metric keys.
	// Can be used for efficient lookup of metrics by name.
	nameBucket map[string][]string
	// tagBucket maps tags to metric keys.
	// Can be used for efficient lookup of metrics by tag.
	tagBucket map[string][]string
	// tagValueBucket maps tags and tag values to metric keys.
	// Can be used for efficient lookup of metrics by tag and value.
	tagValueBucket map[string]map[string][]string
}

// NewMetricsCollector creates a new metrics collector and starts its background processor.
func NewMetricsCollector() MetricsCollector {
	c := &realMetricsCollector{
		queue:   make(chan metricsAction),
		metrics: make(map[string]*Metric),
		nameBucket: make(map[string][]string),
		tagBucket: make(map[string][]string),
		tagValueBucket: make(map[string]map[string][]string),
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
		running = msg.act(c)
	}
}

// Stop stops the collection loop
func (c *realMetricsCollector) Stop() {
	c.queue <- &stopAction{}
}

// Update updates one or more metrics.
// If a metric doesn't exist, it will be created.
// It it exists and has a different type, an error is generated and no update will occur.
// When passing multiple metrics to update, one failure will not prevent other metrics from being updated.
func (c *realMetricsCollector) Update(metrics []Metric, ret chan<- []MetricResponse) {
	c.queue <- &updateAction{metrics, ret}
}

// Fetches one or more metrics.
// Any values passed with the request are ignored.
// If a metric doesn't exist, an error will be generated.
func (c *realMetricsCollector) Fetch(include, exclude MetricFilter, ret chan<- []Metric) {
	c.queue <- &fetchAction{include, exclude, ret}
}
