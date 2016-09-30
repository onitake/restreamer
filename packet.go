import (
	"io"
)

const (
	// TS packet size
	PACKET_SIZE = 188
	// TS packet size with padding
	BUFFER_SIZE = PACKET_SIZE + 4
	// TS packet synchronization byte
	SYNC_BYTE = 0x47
)

// one TS packet
// 188 bytes long
// starts with 0x47
// 4 bytes of padding
// yes, this is a type alias to a byte slice
// use NewPacket to construct a packet, optionally
// by copying data from a data buffer or another packet
type Packet []byte

// creates a new packet
// and optionally fills it with data
func NewPacket(data []byte) Packet {
	// allocate a padded data buffer and create
	// a slice of the correct size from it
	packet := make(Packet, PACKET_SIZE, BUFFER_SIZE)
	if data != nil {
		copy(packet, data)
	}
	return packet
}

// reads data from the input stream,
// scans for the sync byte and returns zero, one or two packets
// the first packet may be partial if it doesn't start with a sync byte
// in that case, the second packet will always be a full one
// if a sync byte can't be found among the next 188 bytes,
// only one packet containing the non-sync data is returned
func ReadPacket(r *Reader) ([]Packet, error) {
	garbage := NewPacket(nil)
	// read 188 bytes ahead (assume we are at the start of a packet)
	_, err := r.Read(p[:PACKET_SIZE])
	if err != nil {
		return []Packet{}, err
	}
	//log.Printf("Read %d bytes\n", n)
	
	// quick check if it starts with the sync byte 0x47
	if garbage[0] != SYNC_BYTE {
		// nope, scan first
		sync := -1
		for i, bytes := range g[:PACKET_SIZE] {
			if bytes == SYNC_BYTE {
				// found, very good
				sync = i
				break
			}
		}
		// nothing found, just return the data
		if sync == -1 {
			return []Packet{garbage}, nil
		}
		// if the sync byte was not at the beginning,
		// create a new resized slice and append the remaining data
		// this should happen only when the stream is out of sync,
		// so performance impact is minimal
		packet := NewPacket(garbage[sync:])
		offset := PACKET_SIZE - sync
		_, err := r.Read(packet[offset:PACKET_SIZE])
		if err != nil {
			return []Packet{garbage[:sync]}, err
		}
		//log.Printf("Appended %d bytes\n", n)
		// return the assembled packet and the remaining data
		return []Packet{packet, garbage[:sync]}, nil
	}
	
	// and done
	return []Packet{packet}, nil
}
