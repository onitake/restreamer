/* Copyright (c) 2018 Gregor Riepl
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

package util

import (
	"testing"
)

func TestBoolean(t *testing.T) {
	a := AtomicFalse
	if a != AtomicFalse {
		t.Error("Invalid AtomicBool value, expected AtomicFalse")
	}
	b := AtomicFalse
	if LoadBool(&b) != false {
		t.Error("Invalid AtomicBool value, expected false")
	}
	c := AtomicTrue
	if c != AtomicTrue {
		t.Error("Invalid AtomicBool value, expected AtomicTrue")
	}
	d := AtomicTrue
	if LoadBool(&d) != true {
		t.Error("Invalid AtomicBool value, expected true")
	}
	var e AtomicBool
	StoreBool(&e, false)
	if LoadBool(&e) != false {
		t.Error("Invalid AtomicBool value, expected false")
	}
	var f AtomicBool
	StoreBool(&f, true)
	if LoadBool(&f) != true {
		t.Error("Invalid AtomicBool value, expected true")
	}
	g := ToAtomicBool(true)
	if LoadBool(&g) != true {
		t.Error("Invalid AtomicBool value, expected true")
	}
}
