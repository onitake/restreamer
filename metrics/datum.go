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
)

var (
	ErrMetricTypeMismatch = errors.New("Cannot update metric of distinct type")
)

// Datum is a generic interface for a metric.
// It represents a single metric datum and supports updating other datums.
//
// Individual implementations should only implement those methods that make
// sense for the data and metric type, and return ErrMetricTypeMismatch for others.
type Datum interface {
	UpdateFrom(Datum) error
	IntGaugeValue() (int64, error)
	FloatGaugeValue() (float64, error)
	BoolGaugeValue() (bool, error)
	StringGaugeValue() (string, error)
	IntCounterValue() (int64, error)
	FloatCounterValue() (float64, error)
}

type int64Gauge int64

func (d *int64Gauge) UpdateFrom(datum Datum) error {
	value, err := datum.IntGaugeValue()
	if err == nil {
		*d = int64Gauge(value)
	}
	return err
}
func (d *int64Gauge) IntGaugeValue() (int64, error) {
	return int64(*d), nil
}
func (d *int64Gauge) FloatGaugeValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}
func (d *int64Gauge) BoolGaugeValue() (bool, error) {
	return false, ErrMetricTypeMismatch
}
func (d *int64Gauge) StringGaugeValue() (string, error) {
	return "", ErrMetricTypeMismatch
}
func (d *int64Gauge) IntCounterValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *int64Gauge) FloatCounterValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}

type float64Gauge float64

func (d *float64Gauge) UpdateFrom(datum Datum) error {
	value, err := datum.FloatGaugeValue()
	if err == nil {
		*d = float64Gauge(value)
	}
	return err
}
func (d *float64Gauge) IntGaugeValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *float64Gauge) FloatGaugeValue() (float64, error) {
	return float64(*d), nil
}
func (d *float64Gauge) BoolGaugeValue() (bool, error) {
	return false, ErrMetricTypeMismatch
}
func (d *float64Gauge) StringGaugeValue() (string, error) {
	return "", ErrMetricTypeMismatch
}
func (d *float64Gauge) IntCounterValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *float64Gauge) FloatCounterValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}

type boolGauge bool

func (d *boolGauge) UpdateFrom(datum Datum) error {
	value, err := datum.BoolGaugeValue()
	if err == nil {
		*d = boolGauge(value)
	}
	return err
}
func (d *boolGauge) IntGaugeValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *boolGauge) FloatGaugeValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}
func (d *boolGauge) BoolGaugeValue() (bool, error) {
	return bool(*d), nil
}
func (d *boolGauge) StringGaugeValue() (string, error) {
	return "", ErrMetricTypeMismatch
}
func (d *boolGauge) IntCounterValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *boolGauge) FloatCounterValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}

type stringGauge string

func (d *stringGauge) UpdateFrom(datum Datum) error {
	value, err := datum.StringGaugeValue()
	if err == nil {
		*d = stringGauge(value)
	}
	return err
}
func (d *stringGauge) IntGaugeValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *stringGauge) FloatGaugeValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}
func (d *stringGauge) BoolGaugeValue() (bool, error) {
	return false, ErrMetricTypeMismatch
}
func (d *stringGauge) StringGaugeValue() (string, error) {
	return string(*d), nil
}
func (d *stringGauge) IntCounterValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *stringGauge) FloatCounterValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}

type int64Counter int64

func (d *int64Counter) UpdateFrom(datum Datum) error {
	value, err := datum.IntCounterValue()
	if err == nil {
		*d += int64Counter(value)
	}
	return err
}
func (d *int64Counter) IntGaugeValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *int64Counter) FloatGaugeValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}
func (d *int64Counter) BoolGaugeValue() (bool, error) {
	return false, ErrMetricTypeMismatch
}
func (d *int64Counter) StringGaugeValue() (string, error) {
	return "", ErrMetricTypeMismatch
}
func (d *int64Counter) IntCounterValue() (int64, error) {
	return int64(*d), nil
}
func (d *int64Counter) FloatCounterValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}

type float64Counter float64

func (d *float64Counter) UpdateFrom(datum Datum) error {
	value, err := datum.FloatCounterValue()
	if err == nil {
		*d += float64Counter(value)
	}
	return err
}
func (d *float64Counter) IntGaugeValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *float64Counter) FloatGaugeValue() (float64, error) {
	return 0.0, ErrMetricTypeMismatch
}
func (d *float64Counter) BoolGaugeValue() (bool, error) {
	return false, ErrMetricTypeMismatch
}
func (d *float64Counter) StringGaugeValue() (string, error) {
	return "", ErrMetricTypeMismatch
}
func (d *float64Counter) IntCounterValue() (int64, error) {
	return 0, ErrMetricTypeMismatch
}
func (d *float64Counter) FloatCounterValue() (float64, error) {
	return float64(*d), nil
}
