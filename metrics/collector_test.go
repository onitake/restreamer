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
	"testing"
)

func TestCollectorCreateStop(t *testing.T) {
	m := NewMetricsCollector()
	m.Stop()
}

func TestCollectorUpdate(t *testing.T) {
	m := NewMetricsCollector()
	u := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: IntGauge(100),
		},
	}
	c := make(chan []MetricResponse)
	m.Update(u, c)
	r := <-c
	if len(r) < 1 || r[0].Error != nil {
		t.Errorf("Expected nil error and value %d, got %v", 100, r)
	} else {
		v, err := r[0].Metric.Value.IntGaugeValue()
		if err != nil {
			t.Errorf("Expected nil error on value get")
		} else {
			if v != 100 {
				t.Errorf("Expected nil error and value %d, got %v", 100, r)
			}
		}
	}
	m.Stop()
}

func TestCollectorUpdateNil(t *testing.T) {
	m := NewMetricsCollector()
	u := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: IntGauge(100),
		},
	}
	m.Update(u, nil)
	m.Stop()
}

func TestCollectorUpdate2Fetch(t *testing.T) {
	m := NewMetricsCollector()
	u := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: IntGauge(100),
		},
	}
	m.Update(u, nil)
	f := []Metric{
		Metric{
			Name:  "TestMetric",
		},
	}
	c := make(chan []MetricResponse)
	m.Fetch(f, c)
	r := <-c
	if len(r) < 1 || r[0].Error != nil {
		t.Errorf("Expected nil error and value %d, got %v", 100, r)
	} else {
		v, err := r[0].Metric.Value.IntGaugeValue()
		if err != nil {
			t.Errorf("Expected nil error on value get")
		} else {
			if v != 100 {
				t.Errorf("Expected nil error and value %d, got %v", 100, r)
			}
		}
	}
	m.Stop()
}

func TestCollector2UpdateFetch(t *testing.T) {
	m := NewMetricsCollector()
	u := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: IntGauge(100),
		},
	}
	m.Update(u, nil)
	u2 := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: IntGauge(200),
		},
	}
	m.Update(u2, nil)
	f := []Metric{
		Metric{
			Name:  "TestMetric",
		},
	}
	c := make(chan []MetricResponse)
	m.Fetch(f, c)
	r := <-c
	if len(r) < 1 || r[0].Error != nil {
		t.Errorf("Expected nil error on set, got %v", r)
	} else {
		v, err := r[0].Metric.Value.IntGaugeValue()
		if err != nil || v != 200 {
				t.Errorf("Expected nil error and value %d, got %d / %v", 200, v, err)
		}
	}
	m.Stop()
}

func TestCollector2UpdateFail(t *testing.T) {
	m := NewMetricsCollector()
	u := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: IntGauge(100),
		},
	}
	m.Update(u, nil)
	u2 := []Metric{
		Metric{
			Name:  "TestMetric",
			Value: FloatGauge(200),
		},
	}
	c := make(chan []MetricResponse)
	m.Update(u2, c)
	r := <-c
	if len(r) < 1 || r[0].Error == nil {
		t.Errorf("Expected error, got %v", r)
	}
	m.Stop()
}
