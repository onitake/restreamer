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
	"bytes"
	"encoding/hex"
)

func TestSlidingWindow01(t *testing.T) {
	w := CreateSlidingWindow(10)
	c := []byte{0,1,2,3}
	w.Put(c)
	r := w.Get()
	x := []byte{0,0,0,0,0,0,0,1,2,3}
	if bytes.Compare(r, x) != 0 {
		t.Errorf("t01: smaller-than-capacity buffer did not compare to padded value:\n%s", hex.Dump(r))
	}
}

func TestSlidingWindow02(t *testing.T) {
	w := CreateSlidingWindow(4)
	c := []byte{0,1,2,3}
	w.Put(c)
	r := w.Get()
	x := []byte{0,1,2,3}
	if bytes.Compare(r, x) != 0 {
		t.Errorf("t01: at-capacity buffer did not compare to the same value:\n%s", hex.Dump(r))
	}
}

func TestSlidingWindow03(t *testing.T) {
	w := CreateSlidingWindow(4)
	c := []byte{0,1,2,3,4,5}
	w.Put(c)
	r := w.Get()
	x := []byte{2,3,4,5}
	if bytes.Compare(r, x) != 0 {
		t.Errorf("t01: over-capacity buffer did not compare to tail:\n%s", hex.Dump(r))
	}
}

func BenchmarkSlidingWindowPutSingle100(b *testing.B) {
	w := CreateSlidingWindow(100)
	for n := 0; n < b.N; n++ {
		w.Put([]byte{0xaa})
	}
}

func BenchmarkSlidingWindowPutMany100(b *testing.B) {
	w := CreateSlidingWindow(100)
	buf := bytes.Repeat([]byte{0xaa}, 50)
	for n := 0; n < b.N; n++ {
		w.Put(buf)
	}
}

func BenchmarkSlidingWindowPutMany100000(b *testing.B) {
	w := CreateSlidingWindow(1000000)
	buf := bytes.Repeat([]byte{0xaa}, 5000)
	for n := 0; n < b.N; n++ {
		w.Put(buf)
	}
}
