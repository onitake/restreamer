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
	"errors"
)

var (
	ErrSequenceQueueEmpty = errors.New("Queue is empty")
	ErrSequenceQueueOutOfBounds = errors.New("Insert index out of bounds")
)

type SequenceQueue struct {
	queue []interface{}
	head int
	tail int
	length int
}

func NewSequenceQueue(bound int) *SequenceQueue {
	return &SequenceQueue{
		queue: make([]interface{}, bound),
	}
}

func (f *SequenceQueue) Length() int {
	return f.length
}

func (f *SequenceQueue) Insert(position int, value interface{}) (old interface{}, err error) {
	if position < 0 || position >= len(f.queue) {
		return nil, ErrSequenceQueueOutOfBounds
	}
	index := (f.head + position) % len(f.queue)
	if position >= f.length {
		f.queue[index] = value
		f.tail = (index + 1) % len(f.queue)
		f.length = position + 1
	} else {
		old = f.queue[index]
		f.queue[index] = value
	}
	return old, nil
}

func (f *SequenceQueue) Peek() (interface{}, error) {
	if f.length == 0 {
		return nil, ErrSequenceQueueEmpty
	}
	return f.queue[f.head], nil
}

func (f *SequenceQueue) Pop() (interface{}, error) {
	if f.length == 0 {
		return nil, ErrSequenceQueueEmpty
	}
	ret := f.queue[f.head]
	f.queue[f.head] = nil
	f.head = (f.head + 1) % len(f.queue)
	f.length--
	return ret, nil
}
