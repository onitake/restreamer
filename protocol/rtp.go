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
	"encoding/binary"
	"errors"
	"io"
)

const (
	DefaultRtpPacketSize int = 1500
	minHeaderSize int = 12
)

var (
	ErrInvalidRtpPacketSize = errors.New("Invalid RTP packet size")
	ErrInvalidRtpVersion = errors.New("Invalid RTP version")
)

type RtpPayloadType uint8
const (
	RtpPayloadTypePCMU RtpPayloadType = 0
	RtpPayloadTypeGSM RtpPayloadType = 3
	RtpPayloadTypeG723 RtpPayloadType = 4
	RtpPayloadTypeDVI4 RtpPayloadType = 5
	RtpPayloadTypeDVI4_2 RtpPayloadType = 6
	RtpPayloadTypeLPC RtpPayloadType = 7
	RtpPayloadTypePCMA RtpPayloadType = 8
	RtpPayloadTypeG722 RtpPayloadType = 9
	RtpPayloadTypeL16 RtpPayloadType = 10
	RtpPayloadTypeL16_2 RtpPayloadType = 11
	RtpPayloadTypeQCELP RtpPayloadType = 12
	RtpPayloadTypeCN RtpPayloadType = 13
	RtpPayloadTypeMPA RtpPayloadType = 14
	RtpPayloadTypeG728 RtpPayloadType = 15
	RtpPayloadTypeDVI4_3 RtpPayloadType = 16
	RtpPayloadTypeDVI4_4 RtpPayloadType = 17
	RtpPayloadTypeG729 RtpPayloadType = 18
	RtpPayloadTypeCelB RtpPayloadType = 25
	RtpPayloadTypeJPEG RtpPayloadType = 26
	RtpPayloadTypeNV RtpPayloadType = 28
	RtpPayloadTypeH261 RtpPayloadType = 31
	RtpPayloadTypeMPV RtpPayloadType = 32
	RtpPayloadTypeMP2T RtpPayloadType = 33
	RtpPayloadTypeH263 RtpPayloadType = 34
)

// RtpPacket represents a decoded RTP packet.
// The header fields are dissected, while the payload is contained as a byte slice.
type RtpPacket struct {
	// Version contains the RTP protocol version.
	// Only version 2 is supported.
	Version uint8
	// Padding is true if the packet hat the padding flag set.
	// The extension header and/or payload is always truncated to the actual size.
	Padding bool
	// Marker is true if the packet hat the marker bit set.
	Marker bool
	// PayloadType describes the kind of data contained in the packet.
	// See the RtpPayloadType constants for known values. Not that some ranges
	// are dynamically assigned and must be defined by the application.
	PayloadType RtpPayloadType
	// SequenceNumber is a 16-bit sequence number that helps with correct ordering
	// wen reassembling the stream.
	SequenceNumber uint16
	// Timestamp is an application-defined absolute timestamp that should be
	// used as the basis for ES timestamps. It can also be ignored if the
	// packaged protocol has sufficiently well-defined timestamps.
	Timestamp uint32
	// Ssrc contains the value of the SSRC field.
	Ssrc uint32
	// Csrc contains the values of the CSRC fields.
	// If the packet had no CSRCs, it is an empty list (nil).
	Csrc []uint32
	// Extension contains the full extension header, including the length and ... fields.
	Extension []byte
	// Payload contains the actual data part of the packet.
	// It is truncated to the packet size, excluding padding (if it was enabled)
	Payload []byte
}

// RtpReader is a packet reader on top of an underlying standard reader.
// It has a configurable maximum packet size.
type RtpReader struct {
	// Reader is the underlying I/O facility
	Reader io.Reader
	// PacketSize is the maximum packet size that can be read.
	// If zero, a default packet size of 1500 octets will be used.
	PacketSize int
}

// ReadRtpPacket reads and returns one RTP packet from the underlying reader.
//
// If the packet was larger than the maximum packet size, excess data will be dropped.
// An incomplete packet, together with a non-nil error will be returned in this case.
//
// If the buffer was to small to read even the RTP header, a nil packet along with
// ErrInvalidRtpPacket will be returned.
//
// If the prtocol version is not equal to 2, ErrInvalidRtpVersion is returned.
func (r *RtpReader) ReadRtpPacket() (*RtpPacket, error) {
	p := &RtpPacket{}

	psize := r.PacketSize
	if psize == 0 {
		psize = DefaultRtpPacketSize
	}
	data := make([]byte, psize)
	n, err := r.Reader.Read(data)
	if n == 0 && err != nil {
		return nil, err
	}
	if n < minHeaderSize {
		return nil, ErrInvalidRtpPacketSize
	}

	p.Version = (data[0] & 0xc0) >> 6
	if p.Version != 2 {
		logger.Logkv(
			"event", "error",
			"error", "rtp_version",
			"version", p.Version,
			"message", "Invalid RTP version",
		)
		return nil, ErrInvalidRtpVersion
	}
	p.Padding = data[0] & 0x20 != 0
	p.Marker = data[1] & 0x80 != 0

	extension := data[0] & 0x10 != 0
	csrcc := int(data[0] & 0x0f)
	xhlen := 0
	if extension {
		xhlen = 4
	}
	if n < minHeaderSize + 4 * csrcc + xhlen {
		return nil, ErrInvalidRtpPacketSize
	}

	p.PayloadType = RtpPayloadType(data[1])
	p.SequenceNumber = binary.BigEndian.Uint16(data[2:4])
	p.Timestamp = binary.BigEndian.Uint32(data[4:8])
	p.Ssrc = binary.BigEndian.Uint32(data[8:12])
	p.Csrc = make([]uint32, csrcc)
	offset := minHeaderSize

	for i := 0; i < csrcc; i++ {
		p.Csrc[i] = binary.BigEndian.Uint32(data[offset:(offset+4)])
		offset += 4
	}

	if extension {
		xlen := int(binary.BigEndian.Uint16(data[(offset+2):(offset+4)]))
		if n < offset + 4 + xlen {
			return nil, ErrInvalidRtpPacketSize
		}
		p.Extension = data[offset:(offset+4+xlen)]
		offset += 4 + xlen
	}

	// TODO truncate padding
	p.Payload = data[offset:n]

	return p, nil
}
