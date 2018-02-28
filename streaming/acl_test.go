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

package streaming

import (
	"testing"
	"github.com/onitake/restreamer/util"
	"sync"
	"sync/atomic"
)

type mockAclLogger struct {
	t *testing.T
	Stage string
}

func (l *mockAclLogger) Log(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
	}
}

func TestAccessController(t *testing.T) {
	l := &mockAclLogger{t, ""}

	l.Stage = "t00"
	c00 := NewAccessController(1)
	c00.SetLogger(l)
	if !c00.Accept("c00b", nil) {
		t.Error("t00: Incorrectly refused connection on free access controller")
	}

	l.Stage = "t01"
	c01 := NewAccessController(1)
	c01.SetLogger(l)
	c01.Accept("", nil)
	if c01.Accept("", nil) {
		t.Error("t01: Incorrectly accepted connection on full access controller")
	}

	l.Stage = "t02"
	c02 := NewAccessController(1)
	c02.SetLogger(l)
	c02.Accept("c02a", nil)
	c02.Release(nil)
	if !c02.Accept("c02b", nil) {
		t.Error("t02: Incorrectly refused connection on freed access controller")
	}

	l.Stage = "t03"
	c03 := NewAccessController(100)
	c03.SetLogger(l)
	w03 := &sync.WaitGroup{}
	w03.Add(100)
	var a03 int32
	for i := 0; i < 100; i++ {
		go func() {
			if c03.Accept("", nil) {
				atomic.AddInt32(&a03, 1)
			}
			w03.Done()
		}()
	}
	w03.Wait()
	if atomic.LoadInt32(&a03) != 100 {
		t.Error("t03: Premature accept failure")
	}
	if c03.Accept("", nil) {
		t.Error("t03: Incorrectly accepted connection on full controller")
	}

	l.Stage = "t04"
	w04 := &sync.WaitGroup{}
	w04.Add(50)
	var a04 int32
	for i := 0; i < 50; i++ {
		go func() {
			c03.Release(nil)
			atomic.AddInt32(&a04, 1)
			w04.Done()
		}()
	}
	w04.Wait()
	if atomic.LoadInt32(&a04) != 50 {
		t.Error("t04: Cannot release half of the connections")
	}

	l.Stage = "t05"
	w05 := &sync.WaitGroup{}
	w05.Add(100)
	var a05 int32
	for i := 0; i < 50; i++ {
		go func() {
			if c03.Accept("", nil) {
				atomic.AddInt32(&a05, 1)
			}
			w05.Done()
		}()
	}
	for i := 0; i < 50; i++ {
		go func() {
			c03.Release(nil)
			atomic.AddInt32(&a05, -1)
			w05.Done()
		}()
	}
	w05.Wait()
	if atomic.LoadInt32(&a05) != 0 {
		t.Error("t05: Cannot release/accept all connections")
	}

	l.Stage = "t06"
	for i := 0; i < 50; i++ {
		if !c03.Accept("", nil) {
			t.Error("t06: Failed to fill up all connection pool")
		}
	}
	if c02.Accept("", nil) {
		t.Error("t06: Incorrectly accepted connection on full controller")
	}
}
