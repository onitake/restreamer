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

package event

import (
	"sync"
	"testing"
	"github.com/onitake/restreamer/util"
)

type mockLogger struct {
	t *testing.T
	Stage string
}

func (l *mockLogger) Log(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
	}
}

type mockLogConnectable struct {
	t *testing.T
	Stage string
	Waiter *sync.WaitGroup
}

func (l *mockLogConnectable) Log(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
		if line["event"] == queueEventConnect {
			l.Waiter.Done()
		}
	}
}

type mockLogDisconnectable struct {
	t *testing.T
	Stage string
	Waiter *sync.WaitGroup
}

func (l *mockLogDisconnectable) Log(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
		if line["event"] == queueEventStopped {
			l.Waiter.Done()
		}
	}
}

type mockHandler struct {
	t *testing.T
	Hit *sync.WaitGroup
	Miss *sync.WaitGroup
}

func (h *mockHandler) HandleEvent(t EventType, args ...interface{}) {
	switch t {
		case EventLimitHit:
			h.Hit.Done()
		case EventLimitMiss:
			h.Miss.Done()
	}
}

func TestCreateLoadReporter(t *testing.T) {
	l := &mockLogger{t, ""}

	// TODO should have timeouts...

	l.Stage = "t00"
	c00 := NewEventQueue(0)
	c00.SetLogger(l)
	c00.Start()
	c00.Shutdown()

	l.Stage = "t01"
	c01 := NewEventQueue(0)
	c01.SetLogger(l)
	c01.Start()
	c01.Start()
	c01.Shutdown()

	c02 := NewEventQueue(0)
	l02 := &mockLogConnectable{
		t,
		"t02",
		&sync.WaitGroup{},
	}
	c02.SetLogger(l02)
	c02.Start()
	l02.Waiter.Add(1)
	c02.NotifyConnect(1)
	l02.Waiter.Wait()
	c02.Shutdown()

	l.Stage = "t03"
	c03 := NewEventQueue(0)
	c03.SetLogger(l)
	c03.Start()
	c03.Shutdown()
	c03.Start()
	c03.Shutdown()

	c04 := NewEventQueue(0)
	l04 := &mockLogConnectable{
		t,
		"t04",
		&sync.WaitGroup{},
	}
	c04.SetLogger(l04)
	c04.Start()
	l04.Waiter.Add(1)
	c04.NotifyConnect(1)
	l04.Waiter.Wait()
	c04.Shutdown()
	c04.Start()
	l04.Waiter.Add(1)
	c04.NotifyConnect(1)
	l04.Waiter.Wait()
	c04.Shutdown()

	c05 := NewEventQueue(10)
	l.Stage = "t05"
	c05.SetLogger(l)
	h05 := &mockHandler{
		t: t,
		Hit: &sync.WaitGroup{},
		Miss: &sync.WaitGroup{},
	}
	h05.Hit.Add(3)
	h05.Miss.Add(2)
	c05.RegisterEventHandler(EventLimitHit, h05)
	c05.RegisterEventHandler(EventLimitMiss, h05)
	c05.Start()
	c05.NotifyConnect(10)
	c05.NotifyConnect(-1)
	c05.NotifyConnect(-2)
	c05.NotifyConnect(4)
	c05.NotifyConnect(1)
	c05.NotifyConnect(-2)
	c05.NotifyConnect(-1)
	c05.NotifyConnect(1)
	h05.Hit.Wait()
	h05.Miss.Wait()
	c05.Shutdown()
}
