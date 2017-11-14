/* Copyright (c) 2016-2017 Gregor Riepl
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
	"io"
	"log"
)

const (
	// PacketSize is the TS packet size (188 bytes)
	PacketSize = 188
	// SyncByte is the byte value of the TS synchronization code (0x47)
	SyncByte = 0x47
)

// Packet is an alias to a byte slice and represents one TS packet.
// It is 188 bytes long and starts with 0x47.
type Packet []byte

// ReadPacket reads data from the input stream,
// scans for the sync byte and returns one packet from that point on.
//
// If a sync byte can't be found among the first 188 bytes,
// no packets are returned
func ReadPacket(reader io.Reader) (Packet, error) {
	garbage := make(Packet, PacketSize)
	offset := 0
	// read 188 bytes ahead (assume we are at the start of a packet)
	for offset < PacketSize {
		nbytes, err := reader.Read(garbage[offset:])
		// read error - bail out
		if err != nil {
			return nil, err
		}
		offset += nbytes
		//log.Printf("Read %d bytes\n", nbytes)
	}

	// quick check if it starts with the sync byte 0x47
	if garbage[0] != SyncByte {
		log.Printf("Partial packet received, scanning for sync byte\n")

		// nope, scan first
		sync := -1
		for i, bytes := range garbage {
			if bytes == SyncByte {
				// found, very good
				sync = i
				break
			}
		}
		// nothing found, return nothing
		if sync == -1 {
			return nil, nil
		}
		//log.Printf("Sync byte found at %d\n", sync)

		offset = 0
		// if the sync byte was not at the beginning,
		// create a new packet and append the remaining data.
		// this should happen only when the stream is out of sync,
		// so performance impact is minimal
		packet := make(Packet, PacketSize)
		copy(packet, garbage[sync:])
		for offset < PacketSize {
			nbytes, err := reader.Read(packet[len(packet):PacketSize])
			if err != nil {
				return nil, err
			}
			offset += nbytes
			//log.Printf("Appended %d bytes\n", nbytes)
		}
		// return the assembled packet
		return packet, nil
	}

	// and done
	return garbage, nil
}
