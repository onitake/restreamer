package restreamer

import (
	"io"
	"log"
)

const (
	// TS packet size
	PacketSize = 188
	// TS packet synchronization byte
	SyncByte = 0x47
)

// one TS packet
// 188 bytes long
// starts with 0x47
// 4 bytes of padding
// yes, this is a type alias to a byte array slice,
// makes everything a bit easier
type Packet []byte

// reads data from the input stream,
// scans for the sync byte and returns one packet from that point on.
// if a sync byte can't be found among the first 188 bytes,
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
