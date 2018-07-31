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
	"github.com/onitake/restreamer/util"
	"sync"
	"testing"
)

type mockLogger struct {
	t     *testing.T
	Stage string
}

func (l *mockLogger) Logd(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
	}
}

func (l *mockLogger) Logkv(keyValues ...interface{}) {
	l.Logd(util.LogFunnel(keyValues))
}

type mockLogConnectable struct {
	t      *testing.T
	Stage  string
	Waiter *sync.WaitGroup
}

func (l *mockLogConnectable) Logd(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
		if line["event"] == queueEventConnect {
			l.Waiter.Done()
		}
	}
}

func (l *mockLogConnectable) Logkv(keyValues ...interface{}) {
	l.Logd(util.LogFunnel(keyValues))
}

type mockLogDisconnectable struct {
	t      *testing.T
	Stage  string
	Waiter *sync.WaitGroup
}

func (l *mockLogDisconnectable) Logd(lines ...util.Dict) {
	for _, line := range lines {
		l.t.Logf("%s: %v", l.Stage, line)
		if line["event"] == queueEventStopped {
			l.Waiter.Done()
		}
	}
}

func (l *mockLogDisconnectable) Logkv(keyValues ...interface{}) {
	l.Logd(util.LogFunnel(keyValues))
}

type mockHandler struct {
	t    *testing.T
	Hit  *sync.WaitGroup
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

func TestCreateLoadReporter00(t *testing.T) {
	l := &mockLogger{t, ""}

	l.Stage = "t00"
	c00 := NewEventQueue(0)
	logger = l
	c00.Start()
	c00.Shutdown()
}

func TestCreateLoadReporter01(t *testing.T) {
	l := &mockLogger{t, ""}

	l.Stage = "t01"
	c01 := NewEventQueue(0)
	logger = l
	c01.Start()
	c01.Start()
	c01.Shutdown()
}

func TestCreateLoadReporter02(t *testing.T) {
	c02 := NewEventQueue(0)
	l02 := &mockLogConnectable{
		t,
		"t02",
		&sync.WaitGroup{},
	}
	logger = l02
	c02.Start()
	l02.Waiter.Add(1)
	c02.NotifyConnect(1)
	l02.Waiter.Wait()
	c02.Shutdown()
}

func TestCreateLoadReporter03(t *testing.T) {
	l := &mockLogger{t, ""}

	l.Stage = "t03"
	c03 := NewEventQueue(0)
	logger = l
	c03.Start()
	c03.Shutdown()
	c03.Start()
	c03.Shutdown()
}

func TestCreateLoadReporter04(t *testing.T) {
	c04 := NewEventQueue(0)
	l04 := &mockLogConnectable{
		t,
		"t04",
		&sync.WaitGroup{},
	}
	logger = l04
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
}

func TestCreateLoadReporter05(t *testing.T) {
	l := &mockLogger{t, ""}

	c05 := NewEventQueue(10)
	l.Stage = "t05"
	logger = l
	h05 := &mockHandler{
		t:    t,
		Hit:  &sync.WaitGroup{},
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
