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

package restreamer

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"testing"
)

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

func testWithProxy(t *testing.T, proxy *Proxy) {
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
	proxy.ServeHTTP(writer, request)
}

func TestProxy(t *testing.T) {
	direct, _ := NewProxy("file:///tmp/test.txt", 10, 0)
	testWithProxy(t, direct)
	cached, _ := NewProxy("file:///tmp/test.txt", 10, 1)
	testWithProxy(t, cached)
}
