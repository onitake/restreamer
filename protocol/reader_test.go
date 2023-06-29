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

package protocol

import (
	"bytes"
	"io"
	"testing"
)

func TestFixedReaderNoData(t *testing.T) {
	d := make([]byte, 0)
	r := bytes.NewBuffer(d)
	f := NewFixedReader(r, 10)
	for i := 0; i < 6; i++ {
		g := make([]byte, 2)
		n, err := f.Read(g)
		if n != 0 || err == nil {
			t.Fatal("Expected 0 bytes and error")
		}
	}
}

func TestFixedReaderNotEnoughInput(t *testing.T) {
	d := make([]byte, 2)
	for i := 0; i < len(d); i++ {
		d[i] = byte(i)
	}
	r := bytes.NewBuffer(d)
	f := NewFixedReader(r, 10)
	g := make([]byte, 2)
	n, err := f.Read(g)
	if n != 2 || err != nil {
		t.Fatal("Expected 2 bytes and no error")
	}
	for i := 0; i < 6; i++ {
		g := make([]byte, 2)
		n, err := f.Read(g)
		if n != 0 || err == nil {
			t.Fatal("Expected 0 bytes and error")
		}
	}
}

func TestFixedReaderJustRight(t *testing.T) {
	d := make([]byte, 10)
	for i := 0; i < len(d); i++ {
		d[i] = byte(i)
	}
	r := bytes.NewBuffer(d)
	f := NewFixedReader(r, 10)
	for i := 0; i < 5; i++ {
		g := make([]byte, 2)
		n, err := f.Read(g)
		if n != 2 || err != nil {
			t.Fatal("Expected 2 bytes and no error")
		}
		if g[0] != byte(i*2) || g[1] != byte(i*2+1) {
			t.Fatal("Expected number sequence")
		}
	}
	g := make([]byte, 2)
	n, err := f.Read(g)
	if n != 0 || err == nil {
		t.Fatal("Expected 0 bytes and error")
	}
}

func TestFixedReaderMultiRead(t *testing.T) {
	d := make([]byte, 20)
	for i := 0; i < len(d); i++ {
		d[i] = byte(i)
	}
	r := bytes.NewBuffer(d)
	f := NewFixedReader(r, 10)
	for i := 0; i < 10; i++ {
		g := make([]byte, 2)
		n, err := f.Read(g)
		if n != 2 || err != nil {
			t.Fatal("Expected 2 bytes and no error")
		}
		if g[0] != byte(i*2) || g[1] != byte(i*2+1) {
			t.Fatal("Expected number sequence")
		}
	}
	g := make([]byte, 2)
	n, err := f.Read(g)
	if n != 0 || err == nil {
		t.Fatal("Expected 0 bytes and error")
	}
}

func TestFixedReaderTailTruncated(t *testing.T) {
	d := make([]byte, 4)
	for i := 0; i < len(d); i++ {
		d[i] = byte(i)
	}
	r := bytes.NewBuffer(d)
	f := NewFixedReader(r, 3)
	g := make([]byte, 2)
	n, err := f.Read(g)
	if n != 2 || err != nil {
		t.Fatal("Expected 2 bytes and no error")
	}
	if g[0] != 0 || g[1] != 1 {
		t.Fatal("Expected number sequence")
	}
	g2 := make([]byte, 2)
	n2, err2 := f.Read(g2)
	if n2 != 1 || err2 != nil {
		t.Fatal("Expected 1 byte and no error")
	}
	if g2[0] != 2 {
		t.Fatal("Expected number sequence")
	}
	g3 := make([]byte, 2)
	n3, err3 := f.Read(g3)
	if n3 != 1 || err3 != nil {
		t.Fatal("Expected 1 byte and no error")
	}
	if g3[0] != 3 {
		t.Fatal("Expected number sequence")
	}
}

type closeableBuffer struct {
	reader io.Reader
	closed bool
}

func (b *closeableBuffer) Read(p []byte) (int, error) {
	if b.closed {
		return 0, io.EOF
	} else {
		return b.reader.Read(p)
	}
}

func (b *closeableBuffer) Close() error {
	b.closed = true
	return nil
}

func TestFixedReaderClose(t *testing.T) {
	d := make([]byte, 4)
	for i := 0; i < len(d); i++ {
		d[i] = byte(i)
	}
	r := &closeableBuffer{bytes.NewBuffer(d), false}
	f := NewFixedReader(r, 2)
	g := make([]byte, 2)
	n, err := f.Read(g)
	if n != 2 || err != nil {
		t.Fatal("Expected 2 bytes and no error")
	}
	if g[0] != 0 || g[1] != 1 {
		t.Fatal("Expected number sequence")
	}
	//goland:noinspection GoUnhandledErrorResult
	f.Close()
	g3 := make([]byte, 2)
	n3, err3 := f.Read(g3)
	if n3 != 0 || err3 == nil {
		t.Fatal("Expected 0 bytes and error")
	}
}

func TestFixedReaderCloseRemain(t *testing.T) {
	d := make([]byte, 6)
	for i := 0; i < len(d); i++ {
		d[i] = byte(i)
	}
	r := &closeableBuffer{bytes.NewBuffer(d), false}
	f := NewFixedReader(r, 4)
	g := make([]byte, 2)
	n, err := f.Read(g)
	if n != 2 || err != nil {
		t.Fatal("Expected 2 bytes and no error")
	}
	if g[0] != 0 || g[1] != 1 {
		t.Fatal("Expected number sequence")
	}
	//goland:noinspection GoUnhandledErrorResult
	f.Close()
	g2 := make([]byte, 2)
	n2, err2 := f.Read(g2)
	if n2 != 2 || err2 != nil {
		t.Fatal("Expected 2 bytes and no error")
	}
	if g2[0] != 2 || g2[1] != 3 {
		t.Fatal("Expected number sequence")
	}
	g3 := make([]byte, 2)
	n3, err3 := f.Read(g3)
	if n3 != 0 || err3 == nil {
		t.Fatal("Expected 0 bytes and error")
	}
}
