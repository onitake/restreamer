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
	"encoding/hex"
	"github.com/onitake/restreamer/auth"
	"github.com/onitake/restreamer/configuration"
	"github.com/onitake/restreamer/util"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type mockProxyLogger struct {
	t      *testing.T
	Closed chan bool
}

func (l *mockProxyLogger) Logd(lines ...util.Dict) {
	shutdown := false
	for _, line := range lines {
		l.t.Logf("%v", line)
		if line["event"] == eventProxyOffline {
			shutdown = true
		}
	}
	if shutdown {
		l.Closed <- true
	}
}

func (l *mockProxyLogger) Logkv(keyValues ...interface{}) {
	l.Logd(util.LogFunnel(keyValues))
}

type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

type MockWriter struct {
	header http.Header
	log    Logger
}

func newMockWriter(logger Logger) *MockWriter {
	return &MockWriter{
		header: make(http.Header),
		log:    logger,
	}
}
func (writer *MockWriter) Header() http.Header {
	return writer.header
}
func (writer *MockWriter) Write(data []byte) (int, error) {
	writer.log.Log("Write data:")
	writer.log.Log(hex.Dump(data))
	return len(data), nil
}
func (writer *MockWriter) WriteHeader(status int) {
	writer.log.Logf("Write header, status code %d:", status)
	writer.log.Log(writer.header)
}

func testWithProxy(t *testing.T, l *mockProxyLogger, proxy *Proxy) {
	logger = l
	writer := newMockWriter(t)
	writer.log = t
	uri, _ := url.ParseRequestURI("http://host/test.txt")
	request := &http.Request{
		Method:     "GET",
		URL:        uri,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     make(http.Header),
	}
	proxy.Start()
	proxy.ServeHTTP(writer, request)
	proxy.Shutdown()
	select {
	case <-l.Closed:
		if len(l.Closed) > 0 {
			t.Fatalf("Multiple shutdown messages received")
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Timeout waiting for proxy shutdown")
	}
}

func TestProxy(t *testing.T) {
	l := &mockProxyLogger{t, make(chan bool)}

	authenticator := auth.NewAuthenticator(configuration.Authentication{}, nil)

	direct, _ := NewProxy("file:///tmp/test.txt", 10, 0, authenticator)
	testWithProxy(t, l, direct)

	cached, _ := NewProxy("file:///tmp/test.txt", 10, 1, authenticator)
	testWithProxy(t, l, cached)
}
