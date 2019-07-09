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

package protocol

import (
	"bytes"
	"io"
)

// FixedReader implements a buffered reader that always reads a fixed amount
// of data from an underlying io.Reader.
//
// It is intended to produce a stream from a packet-based network socket, such
// as net.UDPConn.
//
// Note that packet-based sockets normally give no guarantee about the order of
// incoming packets, and don't account for resends of dropped ones.
// If order is important, reordering must be implemented by other means.
//
// If the underlying reader implements the io.Closer interface, Close() calls
// will be forwarded. Otherwise, Close() is a no-op.
type FixedReader struct {
	reader     io.Reader
	packetSize int
	buffer     *bytes.Buffer
}

// NewPacketBuffer creates a new buffered reader that pulls in data from an
// io.Reader in chunks of psize bytes.
func NewFixedReader(reader io.Reader, psize int) *FixedReader {
	return &FixedReader{
		reader:     reader,
		packetSize: psize,
		buffer:     bytes.NewBuffer(make([]byte, 0, psize)),
	}
}

// Read reads as many bytes from the internal buffer as can fit into p.
//
// If the buffer has no data left, it tries to pull in a new packet from the
// underlying reader.
func (b *FixedReader) Read(p []byte) (n int, err error) {
	// check if we need to read another packet
	if b.buffer.Len() == 0 {
		// read the next packet
		p := make([]byte, b.packetSize)
		var m int
		// pass on err if the read fails
		m, err = b.reader.Read(p)
		// only buffer as many bytes as were received
		b.buffer.Write(p[:m])
	}
	if err == nil {
		// if the was no I/O error, pass on any buffer errors
		n, err = b.buffer.Read(p)
	} else {
		// ignore buffer errors and pass on the I/O error instead
		n, _ = b.buffer.Read(p)
	}
	return n, err
}

// Close closes the underlying reader.
//
// Subsequent Read calls will succeed as long as the internal buffer still
// has data. If the buffer is drained, Read returns an error.
func (b *FixedReader) Close() error {
	if closer, ok := b.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
