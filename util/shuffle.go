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
	"math/rand"
)

// Shuffle shuffles a slice using Knuth's version of the Fisher-Yates algorithm.
func ShuffleStrings(rnd *rand.Rand, list []string) []string {
	N := len(list)
	ret := make([]string, N)
	copy(ret, list)
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rnd.Intn(N-i)
		ret[r], ret[i] = ret[i], ret[r]
	}
	return ret
}
