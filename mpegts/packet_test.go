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

package mpegts

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func makeRandomPacket(t *testing.T, bytes int) []byte {
	packet := make([]byte, bytes)
	rnd := rand.New(rand.NewSource(18847))
	if n, err := rnd.Read(packet); err != nil || n != bytes {
		t.Fatal("Error creating a random packet")
	}
	return packet
}

func TestPacketScan(t *testing.T) {
	c01 := bytes.NewBuffer([]byte{})
	r01, err := ReadPacket(c01)
	if err != io.EOF || r01 != nil {
		t.Error("t01: Expected EOF on empty buffer, got something else")
	}

	c02 := bytes.NewBuffer([]byte{
		0x00,
	})
	r02, err := ReadPacket(c02)
	if err != io.EOF || r02 != nil {
		t.Error("t02: Expected EOF on incomplete buffer without sync, got something else")
	}

	c03 := bytes.NewBuffer([]byte{
		0x47, 0x00,
	})
	r03, err := ReadPacket(c03)
	if err != io.EOF || r03 != nil {
		t.Error("t03: Expected EOF on incomplete buffer, got something else")
	}

	c04 := bytes.NewBuffer(make([]byte, 188))
	r04, err := ReadPacket(c04)
	if err != nil || len(r04) != 0 {
		t.Errorf("t04: Expected empty result on zero buffer, got something else")
	}

	b05 := makeRandomPacket(t, 188)
	b05[0] = 0x47
	c05 := bytes.NewBuffer(b05)
	r05, err := ReadPacket(c05)
	if err != nil || !bytes.Equal(b05, r05) {
		t.Error("t05: Expected packet identical to buffer, got an error or something else")
	}

	b06 := makeRandomPacket(t, 189)
	b06[0] = 0x47
	c06 := bytes.NewBuffer(b06)
	r06, err := ReadPacket(c06)
	if err != nil || !bytes.Equal(b06[:188], r06) {
		t.Error("t06: Expected packet identical to head of buffer, got an error or something else")
	}

	b07 := makeRandomPacket(t, 189)
	b07[0] = 0
	b07[1] = 0x47
	c07 := bytes.NewBuffer(b07)
	r07, err := ReadPacket(c07)
	if err != nil || !bytes.Equal(b07[1:189], r07) {
		t.Error("t07: Expected packet identical to tail of buffer, got an error or something else")
	}

	b08 := makeRandomPacket(t, 188)
	b08[0] = 0
	b08[1] = 0x47
	c08 := bytes.NewBuffer(b08)
	r08, err := ReadPacket(c08)
	if err != io.EOF || r08 != nil {
		t.Error("t08: Expected EOF on incomplete packet that didn't start at offset 0, got something else")
	}
}
