/* Copyright (c) 2016-2018 Gregor Riepl
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
	"io"
)

const (
	// MpegTsPacketSize is the TS packet size (188 bytes)
	MpegTsPacketSize = 188
	// SyncByte is the byte value of the TS synchronization code (0x47)
	MpegTsSyncByte = 0x47
)

// MpegTsPacket is an alias to a byte slice and represents one TS packet.
// It is 188 bytes long and starts with 0x47.
type MpegTsPacket []byte

// ReadPacket reads data from the input stream,
// scans for the sync byte and returns one packet from that point on.
//
// If a sync byte can't be found among the first 188 bytes,
// no packets are returned
func ReadMpegTsPacket(reader io.Reader) (MpegTsPacket, error) {
	garbage := make(MpegTsPacket, MpegTsPacketSize)
	offset := 0
	// read 188 bytes ahead (assume we are at the start of a packet)
	for offset < MpegTsPacketSize {
		nbytes, err := reader.Read(garbage[offset:])
		// read error - bail out
		if err != nil {
			return nil, err
		}
		offset += nbytes
		//logger.Logkv("event", "read", "bytes", nbytes)
	}

	// quick check if it starts with the sync byte 0x47
	if garbage[0] != MpegTsSyncByte {
		//logger.Logkv("event", "partial")

		// nope, scan first
		sync := -1
		for i, bytes := range garbage {
			if bytes == MpegTsSyncByte {
				// found, very good
				sync = i
				break
			}
		}
		// nothing found, return nothing
		if sync == -1 {
			return nil, nil
		}
		//logger.Logkv("event", "sync", "position", sync)

		// if the sync byte was not at the beginning,
		// create a new packet and append the remaining data.
		// this should happen only when the stream is out of sync,
		// so performance impact is minimal
		packet := make(MpegTsPacket, MpegTsPacketSize)
		offset = len(packet) - sync
		//logger.Logkv("event", "offset", "offset", offset)
		copy(packet, garbage[sync:])
		for offset < MpegTsPacketSize {
			nbytes, err := reader.Read(packet[offset:])
			if err != nil {
				return nil, err
			}
			offset += nbytes
			//logger.Logkv("event", "append", "bytes", nbytes, "position", offset)
		}
		// return the assembled packet
		return packet, nil
	}

	// and done
	return garbage, nil
}
