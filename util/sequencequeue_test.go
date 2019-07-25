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

func TestSequenceQueuePush(t *testing.T) {
	q := NewSequenceQueue(1)
	e := q.Insert(0, "A")
	if e != nil {
		t.Errorf("Insert returned error: %v", e)
	}
	l := q.Length()
	if l != 1 {
		t.Errorf("Length after insert is %d instead of %d", l, 1)
	}
	if l != q.Length() {
		t.Errorf("Second length call mismatch")
	}
}

func TestSequenceQueuePushOob(t *testing.T) {
	q := NewSequenceQueue(1)
	e := q.Insert(1, "A")
	if e != ErrSequenceQueueOutOfBounds {
		t.Errorf("Didn't get out of bounds error but: %v", e)
	}
	e = q.Insert(-1, "B")
	if e != ErrSequenceQueueOutOfBounds {
		t.Errorf("Didn't get out of bounds error but: %v", e)
	}
	l := q.Length()
	if l != 0 {
		t.Errorf("Length after insert is %d instead of %d", l, 0)
	}
}

func TestSequenceQueuePushPop(t *testing.T) {
	q := NewSequenceQueue(1)
	e := q.Insert(0, "A")
	if e != nil {
		t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
		t.Errorf("Insert returned error: %v", e)
	}
	r, e := q.Pop()
	if e != nil {
		t.Fatalf("Got error: %v", e)
	}
	if r != "A" {
		t.Errorf("Got value: %v", r)
		t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
	}
}

func TestSequenceQueuePushOccupied(t *testing.T) {
	q := NewSequenceQueue(1)
	e := q.Insert(0, "A")
	if e != nil {
		t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
		t.Errorf("Insert returned error: %v", e)
	}
	e = q.Insert(0, "A")
	if e != ErrSequenceQueueOccupied {
		t.Fatalf("Got nil instead of error")
	}
}

func TestSequenceQueuePushMutiple(t *testing.T) {
	q := NewSequenceQueue(10)
	for i := 0; i < 10; i++ {
		e := q.Insert(i, i)
		if e != nil {
			t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
			t.Errorf("Insert returned error: %v", e)
		}
	}
	for i := 0; i < 10; i++ {
		r, e := q.Pop()
		if e != nil {
			t.Fatalf("Got error: %v", e)
		}
		if r != i {
			t.Errorf("Got value: %v expected: %v", r, i)
			t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
		}
	}
	r, e := q.Pop()
	if e != ErrSequenceQueueEmpty || r != nil {
		t.Fatalf("Expected empty queue, got: %v value=%v", e, r)
	}
}

func TestSequenceQueuePushReverse(t *testing.T) {
	q := NewSequenceQueue(10)
	for i := 9; i >= 0; i-- {
		e := q.Insert(i, i)
		if e != nil {
			t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
			t.Errorf("Insert returned error: %v", e)
		}
		if i != 0 {
			r, e := q.Pop()
			if e != ErrSequenceQueueSlotEmpty {
				t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
				t.Fatalf("Expected empty slot at head, got: %v", e)
			}
			if r != nil {
				t.Errorf("Got value: %v expected: %v", r, nil)
			}
		}
	}
	for i := 0; i < 10; i++ {
		r, e := q.Pop()
		if e != nil {
			t.Fatalf("Got error: %v", e)
		}
		if r != i {
			t.Errorf("Got value: %v expected: %v", r, i)
			t.Logf("head=%d tail=%d length=%d", q.head, q.tail, q.length)
		}
	}
	r, e := q.Pop()
	if e != ErrSequenceQueueEmpty {
		t.Fatalf("Expected empty queue, got: %v value=%v", e, r)
	}
	if r != nil {
		t.Errorf("Got value: %v expected: %v", r, nil)
	}
}
