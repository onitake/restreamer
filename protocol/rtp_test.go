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
	//"io"
	"testing"
)

func listEqual(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func comparePackets(t *testing.T, x, g *RtpPacket) {
	if x == nil && g != nil {
		t.Error("Expected a nil packet, got non-nil")
	}
	if x != nil && g == nil {
		t.Error("Expected a non-nil packet, got nil")
	}
	if x.Version != g.Version {
		t.Errorf("Got incorrect version: %d Expected: %d", g.Version, x.Version)
	}
	if x.Padding != g.Padding {
		t.Errorf("Got incorrect padding flag: %v Expected: %v", g.Padding, x.Padding)
	}
	if x.Marker != g.Marker {
		t.Errorf("Got incorrect marker flag: %v Expected: %v", g.Marker, x.Marker)
	}
	if x.PayloadType != g.PayloadType {
		t.Errorf("Got incorrect payload type: %v Expected: %v", g.PayloadType, x.PayloadType)
	}
	if x.SequenceNumber != g.SequenceNumber {
		t.Errorf("Got incorrect sequence number: %v Expected: %v", g.SequenceNumber, x.SequenceNumber)
	}
	if x.Timestamp != g.Timestamp {
		t.Errorf("Got incorrect timestamp: %v Expected: %v", g.Timestamp, x.Timestamp)
	}
	if x.Ssrc != g.Ssrc {
		t.Errorf("Got incorrect SSRC: %v Expected: %v", g.Ssrc, x.Ssrc)
	}
	if !listEqual(x.Csrc, g.Csrc) {
		t.Errorf("CSRC doesn't match: %v Expected: %v", g.Csrc, x.Csrc)
	}
	if !bytes.Equal(x.Extension, g.Extension) {
		t.Errorf("Extension doesn't match: %v Expected: %v", g.Extension, x.Extension)
	}
	if !bytes.Equal(x.Payload, g.Payload) {
		t.Errorf("Payload doesn't match: %v Expected: %v", g.Payload, x.Payload)
	}
}

func TestRtpEmptyPacket(t *testing.T) {
	d := []byte{}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if s != nil || err == nil {
		t.Fatalf("Expected non-nil error and nil packet")
	}
}

func TestRtpIncompleteHeader(t *testing.T) {
	d := []byte{0x02, 0x21, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	for i := 1; i < 11; i++ {
		b := bytes.NewBuffer(d[0:i])
		r := RtpReader{
			Reader: b,
		}
		s, err := r.ReadRtpPacket()
		if s != nil || err != ErrInvalidRtpPacketSize {
			t.Fatalf("Expected non-nil error and nil packet: %v", err)
		}
	}
}

func TestRtpInvalidVersion(t *testing.T) {
	d := []byte{0x01, 0x21, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if s != nil || err != ErrInvalidRtpVersion {
		t.Fatalf("Expected non-nil error and nil packet: %v", err)
	}
}

func TestRtpMissingCsrc(t *testing.T) {
	d := []byte{0x12, 0x21, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if s != nil || err != ErrInvalidRtpPacketSize {
		t.Fatalf("Expected non-nil error and nil packet: %v", err)
	}
}

func TestRtpMissingExtension(t *testing.T) {
	d := []byte{0x0e, 0x21, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if s != nil || err != ErrInvalidRtpPacketSize {
		t.Fatalf("Expected non-nil error and nil packet: %v", err)
	}
}

func TestRtpIncompleteExtension(t *testing.T) {
	d := []byte{0x0e, 0x21, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x22, 0x00, 0x04}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if s != nil || err != ErrInvalidRtpPacketSize {
		t.Fatalf("Expected non-nil error and nil packet: %v", err)
	}
}

func TestRtpHeaderOnly(t *testing.T) {
	d := []byte{0x02, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: false,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: nil,
		Extension: nil,
		Payload: nil,
	}
	comparePackets(t, x, s)
}

func TestRtpHeaderCsrc(t *testing.T) {
	d := []byte{0x26, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x10, 0x0e, 0x20, 0x0d, 0x40, 0x0b, 0x80, 0x07}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: true,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: []uint32{
			0x100e200d,
			0x400b8007,
		},
		Extension: nil,
		Payload: nil,
	}
	comparePackets(t, x, s)
}

func TestRtpHeaderExtension(t *testing.T) {
	d := []byte{0x0e, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: true,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: nil,
		Extension: []byte{
			0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7,
		},
		Payload: nil,
	}
	comparePackets(t, x, s)
}

func TestRtpHeaderCsrcExtension(t *testing.T) {
	d := []byte{0x2e, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x10, 0x0e, 0x20, 0x0d, 0x40, 0x0b, 0x80, 0x07, 0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: true,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: []uint32{
			0x100e200d,
			0x400b8007,
		},
		Extension: []byte{
			0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7,
		},
		Payload: nil,
	}
	comparePackets(t, x, s)
}

func TestRtpPayload(t *testing.T) {
	d := []byte{0x02, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: false,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: nil,
		Extension: nil,
		Payload: []byte{
			0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70,
		},
	}
	comparePackets(t, x, s)
}

func TestRtpCsrcPayload(t *testing.T) {
	d := []byte{0x26, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x10, 0x0e, 0x20, 0x0d, 0x40, 0x0b, 0x80, 0x07, 0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: true,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: []uint32{
			0x100e200d,
			0x400b8007,
		},
		Extension: nil,
		Payload: []byte{
			0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70,
		},
	}
	comparePackets(t, x, s)
}

func TestRtpExtensionPayload(t *testing.T) {
	d := []byte{0x0e, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7, 0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: true,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: nil,
		Extension: []byte{
			0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7,
		},
		Payload: []byte{
			0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70,
		},
	}
	comparePackets(t, x, s)
}

func TestRtpCsrcExtensionPayload(t *testing.T) {
	d := []byte{0x2e, 0x21, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xaa, 0x55, 0x10, 0x0e, 0x20, 0x0d, 0x40, 0x0b, 0x80, 0x07, 0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7, 0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70}
	b := bytes.NewBuffer(d)
	r := RtpReader{
		Reader: b,
	}
	s, err := r.ReadRtpPacket()
	if err != nil {
		t.Fatalf("Got an error instead of a packet: %v", err)
	}
	x := &RtpPacket{
		Version: 2,
		Padding: true,
		Marker: false,
		PayloadType: RtpPayloadTypeMP2T,
		SequenceNumber: 0x0123,
		Timestamp: 0x456789ab,
		Ssrc: 0xcdefaa55,
		Csrc: []uint32{
			0x100e200d,
			0x400b8007,
		},
		Extension: []byte{
			0x11, 0xff, 0x00, 0x08, 0x15, 0xae, 0x25, 0xad, 0x45, 0xab, 0x85, 0xa7,
		},
		Payload: []byte{
			0x01, 0x02, 0x03, 0x04, 0x50, 0x60, 0x70,
		},
	}
	comparePackets(t, x, s)
}
