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

// AbsSubInt32 calculates absolute difference between a and b, or: | a - b |
// Arguments and return type are int32.
func AbsSubInt32(a, b int32) int32 {
	if a > b {
		return a - b
	}
	return b - a
}

// AbsSubInt64 calculates absolute difference between a and b, or: | a - b |
// Arguments and return type are int64.
func AbsSubInt64(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

// AbsSub calculates absolute difference between a and b, or: | a - b |
// Arguments and return type are int.
func AbsSub(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
