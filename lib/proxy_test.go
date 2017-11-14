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
	"log"
	"net/http"
	"net/url"
	"testing"
)

type testWriter struct {
	header http.Header
}

func newTestWriter() *testWriter {
	return &testWriter{
		header: make(http.Header),
	}
}

func (writer *testWriter) Header() http.Header {
	return writer.header
}

func (writer *testWriter) Write(data []byte) (int, error) {
	log.Printf("Write data:")
	log.Print(hex.Dump(data))
	return len(data), nil
}

func (writer *testWriter) WriteHeader(status int) {
	log.Printf("Write header, status code %d:", status)
	log.Print(writer.header)
}

func TestDirect(t *testing.T) {
	direct, _ := NewProxy("file:///tmp/test.txt", 10, 0)

	writer := newTestWriter()
	uri, _ := url.ParseRequestURI("http://host/test.txt")
	request := &http.Request{
		Method:     "GET",
		URL:        uri,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     make(http.Header),
	}
	direct.ServeHTTP(writer, request)
}

func TestCached(t *testing.T) {
	cached, _ := NewProxy("file:///tmp/test.txt", 10, 1)
	writer := newTestWriter()
	uri, _ := url.ParseRequestURI("http://host/test.txt")
	request := &http.Request{
		Method:     "GET",
		URL:        uri,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     make(http.Header),
	}
	cached.ServeHTTP(writer, request)
}
