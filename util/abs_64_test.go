// +build amd64 arm64 ppc64

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

package util

import (
	"testing"
)

func TestAbsSubInt32_64(t *testing.T) {
	va := []int32{ -0x7ffffffe, 0x7ffffffe, -0x7fffffff, 0x7fffffff, -2, 2, -1, -1, 0, 0 }
	vb := []int32{ -0x7ffffffe, 0x7ffffffe, -0x7fffffff, 0x7fffffff, -2, 2, -1, -1, 0, 0 }
	for ia := 0; ia < len(va); ia++ {
		for ib := 0; ib < len(vb); ib++ {
			a := va[ia]
			b := vb[ib]
			r := AbsSubInt32(a, b)
			var x int32
			if a > b {
				x = a - b
			} else {
				x = b - a
			}
			if r != x {
				t.Errorf("AbsSub(%d, %d) should be %d, got %d", a, b, x, r)
			}
		}
	}
}

func TestAbsSubInt64_64(t *testing.T) {
	va := []int64{ -0x7ffffffffffffffe, 0x7ffffffffffffffe, -0x7fffffffffffffff, 0x7fffffffffffffff, -0x7ffffffe, 0x7ffffffe, -0x7fffffff, 0x7fffffff, -2, 2, -1, -1, 0, 0 }
	vb := []int64{ -0x7ffffffffffffffe, 0x7ffffffffffffffe, -0x7fffffffffffffff, 0x7fffffffffffffff, -0x7ffffffe, 0x7ffffffe, -0x7fffffff, 0x7fffffff, -2, 2, -1, -1, 0, 0 }
	for ia := 0; ia < len(va); ia++ {
		for ib := 0; ib < len(vb); ib++ {
			a := va[ia]
			b := vb[ib]
			r := AbsSubInt64(a, b)
			var x int64
			if a > b {
				x = a - b
			} else {
				x = b - a
			}
			if r != x {
				t.Errorf("AbsSub(%d, %d) should be %d, got %d", a, b, x, r)
			}
		}
	}
}

func TestAbsSub_64(t *testing.T) {
	va := []int{  -0x7ffffffffffffffe, 0x7ffffffffffffffe, -0x7fffffffffffffff, 0x7fffffffffffffff, -0x7ffffffe, 0x7ffffffe, -0x7fffffff, 0x7fffffff, -2, 2, -1, -1, 0, 0 }
	vb := []int{ -0x7ffffffffffffffe, 0x7ffffffffffffffe, -0x7fffffffffffffff, 0x7fffffffffffffff, -0x7ffffffe, 0x7ffffffe, -0x7fffffff, 0x7fffffff, -2, 2, -1, -1, 0, 0 }
	for ia := 0; ia < len(va); ia++ {
		for ib := 0; ib < len(vb); ib++ {
			a := va[ia]
			b := vb[ib]
			r := AbsSub(a, b)
			var x int
			if a > b {
				x = a - b
			} else {
				x = b - a
			}
			if r != x {
				t.Errorf("AbsSub(%d, %d) should be %d, got %d", a, b, x, r)
			}
		}
	}
}
