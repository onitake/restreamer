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

// Set is a set type, based on a map where keys represent the values.
//
// You can use both map semantics or the provided convenience functions
// for basic set operations.
//
// Map semantics:
//   Create:
//     set := make(Set)
//   Add:
//     set[value] = true
//   Remove:
//     delete(set, value)
//   Test:
//     if set[value] { }
//
// Convenience functions:
//   Create:
//     set := MakeSet()
//   Add:
//     set.Add(value)
//   Remove:
//     set.Remove(value)
//   Test:
//     if set.Contains(value)
type Set map[interface{}] bool

// MakeSet creates a new empty set.
func MakeSet() Set {
	return make(Set)
}

// Add adds a value to the set.
func (set Set) Add(value interface{}) {
	set[value] = true
}

// Remove removes a value from the set.
func (set Set) Remove(value interface{}) {
	delete(set, value)
}

// Contains tests if a value is in the set and returns true if this is the case.
func (set Set) Contains(value interface{}) bool {
	return set[value]
}
