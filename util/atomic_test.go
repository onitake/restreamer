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

func TestValue(t *testing.T) {
	var z AtomicBool
	if z != AtomicFalse {
		t.Error("Invalid AtomicBool default value, expected AtomicFalse")
	}
	a := AtomicFalse
	if a != AtomicFalse {
		t.Error("Invalid AtomicBool value, expected AtomicFalse")
	}
	c := AtomicTrue
	if c != AtomicTrue {
		t.Error("Invalid AtomicBool value, expected AtomicTrue")
	}
}

func TestLoad(t *testing.T) {
	b := AtomicFalse
	if LoadBool(&b) != false {
		t.Error("Invalid AtomicBool value, expected false")
	}
	d := AtomicTrue
	if LoadBool(&d) != true {
		t.Error("Invalid AtomicBool value, expected true")
	}
}

func TestLoadStore(t *testing.T) {
	var e AtomicBool
	StoreBool(&e, false)
	if e != AtomicFalse || LoadBool(&e) != false {
		t.Error("Invalid AtomicBool value, expected false")
	}
	var f AtomicBool
	StoreBool(&f, true)
	if f != AtomicTrue || LoadBool(&f) != true {
		t.Error("Invalid AtomicBool value, expected true")
	}
}

func TestConvert(t *testing.T) {
	g := ToAtomicBool(false)
	if g != AtomicFalse || LoadBool(&g) != false {
		t.Error("Invalid AtomicBool value, expected false")
	}
	h := ToAtomicBool(true)
	if h != AtomicTrue || LoadBool(&h) != true {
		t.Error("Invalid AtomicBool value, expected true")
	}
}

func TestCompareAndSwap(t *testing.T) {
	i := AtomicFalse
	if !CompareAndSwapBool(&i, false, false) || i != AtomicFalse {
		t.Error("Invalid AtomicBool value, expected swap with false")
	}
	j := AtomicFalse
	if !CompareAndSwapBool(&j, false, true) || j != AtomicTrue {
		t.Error("Invalid AtomicBool value, expected swap with true")
	}
	k := AtomicFalse
	if CompareAndSwapBool(&k, true, false) || k != AtomicFalse {
		t.Error("Invalid AtomicBool value, expected no swap with false")
	}
	l := AtomicFalse
	if CompareAndSwapBool(&l, true, true) || l != AtomicFalse {
		t.Error("Invalid AtomicBool value, expected no swap with true")
	}
	m := AtomicTrue
	if CompareAndSwapBool(&m, false, false) || m != AtomicTrue {
		t.Error("Invalid AtomicBool value, expected no swap with false")
	}
	n := AtomicTrue
	if CompareAndSwapBool(&n, false, true) || n != AtomicTrue {
		t.Error("Invalid AtomicBool value, expected no swap with true")
	}
	o := AtomicTrue
	if !CompareAndSwapBool(&o, true, false) || o != AtomicFalse {
		t.Error("Invalid AtomicBool value, expected swap with false")
	}
	p := AtomicTrue
	if !CompareAndSwapBool(&p, true, true) || p != AtomicTrue {
		t.Error("Invalid AtomicBool value, expected swap with true")
	}
}

func TestSwap(t *testing.T) {
	q := AtomicFalse
	if SwapBool(&q, false) != false && q == AtomicFalse {
		t.Error("Invalid AtomicBool value, expected false,false")
	}
	r := AtomicFalse
	if SwapBool(&r, true) != false && r == AtomicTrue {
		t.Error("Invalid AtomicBool value, expected false,true")
	}
	s := AtomicTrue
	if SwapBool(&s, false) != true && s == AtomicFalse {
		t.Error("Invalid AtomicBool value, expected true,false")
	}
	u := AtomicTrue
	if SwapBool(&u, true) != true && u == AtomicTrue {
		t.Error("Invalid AtomicBool value, expected true,true")
	}
}
