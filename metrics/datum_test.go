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

func TestDatumIntGauge(t *testing.T) {
	s := int64Gauge(100)
	ps := &s
	i := int64Gauge(200)
	pi := &i
	if err := pi.UpdateFrom(ps); err != nil || *ps != int64Gauge(100) || *pi != int64Gauge(100) {
		t.Errorf("Expected successful update to %v, got %v, source %v, err %v", 100, *pi, *ps, err)
	}
	f := float64Gauge(300.0)
	pf := &f
	if err := pf.UpdateFrom(ps); err == nil {
		t.Errorf("Expected type mismatch error, got %v", ErrMetricTypeMismatch)
	}
	b := boolGauge(true)
	pb := &b
	if err := pb.UpdateFrom(ps); err == nil {
		t.Errorf("Expected type mismatch error, got %v", ErrMetricTypeMismatch)
	}
	r := stringGauge("hello world")
	pr := &r
	if err := pr.UpdateFrom(ps); err == nil {
		t.Errorf("Expected type mismatch error, got %v", ErrMetricTypeMismatch)
	}
	ci := int64Counter(400)
	pci := &ci
	if err := pci.UpdateFrom(ps); err == nil {
		t.Errorf("Expected type mismatch error, got %v", ErrMetricTypeMismatch)
	}
	cf := int64Counter(500.0)
	pcf := &cf
	if err := pcf.UpdateFrom(ps); err == nil {
		t.Errorf("Expected type mismatch error, got %v", ErrMetricTypeMismatch)
	}
}
