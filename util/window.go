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
	"sync"
)

// SlidingWindow implements a buffer that continuously overwrites old data.
// A single fetch function that copies the whole window is provided.
//
// Note that the implementation uses a variable buffer and may not execute
// in deterministic time.
//
// All operations are thread-safe.
type SlidingWindow struct {
	window  []byte
	lock    sync.RWMutex
}

// SlidingWindow creates a sliding window buffer with a fixed size.
// Note that the buffer is pre-filled with 0s.
func CreateSlidingWindow(size int) *SlidingWindow {
	return &SlidingWindow{
		window: make([]byte, size),
	}
}

// Put copies the contents of data into the sliding window buffer.
// If data is longer than the buffer, the head will be cut off until it fits.
func (w *SlidingWindow) Put(data []byte) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.window = append(w.window, data...)[len(data):]
}

// Get returns the contents of the sliding window buffer.
// No copying is performed, the return value is simply a slice of th e buffer.
// Take care not to modify the contents.
func (w *SlidingWindow) Get() []byte {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.window
}
