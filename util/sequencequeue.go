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
	ErrSequenceQueueOccupied = errors.New("Queue position is already occupied")
	ErrSequenceQueueOutOfBounds = errors.New("Insert index out of bounds")
	ErrSequenceQueueSlotEmpty = errors.New("Queue slot is empty")
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

func (f *SequenceQueue) Insert(position int, value interface{}) error {
	if position < 0 || position >= len(f.queue) {
		return ErrSequenceQueueOutOfBounds
	}
	index := (f.head + position) % len(f.queue)
	if position >= f.length {
		f.queue[index] = value
		f.tail = (index + 1) % len(f.queue)
		f.length = position + 1
	} else {
		if f.queue[index] == nil {
			f.queue[index] = value
		} else {
			return ErrSequenceQueueOccupied
		}
	}
	return nil
}

func (f *SequenceQueue) Pop() (interface{}, error) {
	if f.length == 0 {
		return nil, ErrSequenceQueueEmpty
	}
	ret := f.queue[f.head]
	if ret == nil {
		return nil, ErrSequenceQueueSlotEmpty
	}
	f.queue[f.head] = nil
	f.head = (f.head + 1) % len(f.queue)
	f.length--
	return ret, nil
}
