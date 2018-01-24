/* Copyright (c) 2017 Gregor Riepl
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
	"sync/atomic"
)

// AtomicBool is a type placeholder for atomic operations on booleans.
//
// It shares its type with int32 and relies on the atomic.*Int32 functions
// to implement the underlying atomic operations.
//
// CompareAndSwapBool, LoadBool, StoreBool and SwapBool are provided as
// convenience frontends. They automatically cast between int32 and bool
// types and should hide the nitty-gritty details.
type AtomicBool int32

const (
	AtomicFalse AtomicBool = 0
	AtomicTrue  AtomicBool = 1
)

// ToAtomicBool converts a bool value to the corresponding AtomicBool value
func ToAtomicBool(value bool) AtomicBool {
	if value {
		return AtomicTrue
	} else {
		return AtomicFalse
	}
}

// CompareAndSwapBool executes the compare-and-swap operation for a boolean value.
func CompareAndSwapBool(addr *AtomicBool, old, new bool) (swapped bool) {
	var o int32
	if old {
		o = int32(AtomicTrue)
	}
	var n int32
	if new {
		n = int32(AtomicTrue)
	}
	return atomic.CompareAndSwapInt32((*int32)(addr), o, n)
}

// LoadBool atomically loads *addr.
func LoadBool(addr *AtomicBool) (val bool) {
	var r int32 = atomic.LoadInt32((*int32)(addr))
	return r != int32(AtomicFalse)
}

// StoreBool atomically stores val into *addr.
func StoreBool(addr *AtomicBool, val bool) {
	var v int32
	if val {
		v = int32(AtomicTrue)
	}
	atomic.StoreInt32((*int32)(addr), v)
}

// SwapBool atomically stores new into *addr and returns the previous *addr value.
func SwapBool(addr *AtomicBool, new bool) (old bool) {
	var n int32
	if new {
		n = int32(AtomicTrue)
	}
	var r int32 = atomic.SwapInt32((*int32)(addr), n)
	return r != int32(AtomicFalse)
}
