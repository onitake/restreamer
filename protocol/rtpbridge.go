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
	"errors"
	"io"
	"github.com/onitake/restreamer/util"
	"fmt"
)

var (
	ErrRtpBridgeInvalidPacketType = errors.New("Invalid packet type")
	ErrRtpBridgePacketTooSmall = errors.New("Payload too small")
	ErrRtpBridgeWaitingForMore = errors.New("Waiting for more data")
)

type RtpBridge struct {
	reader *RtpReader
	seqnum int
	queue  *util.SequenceQueue
	buffer *bytes.Buffer
}

func NewRtpBridge(reader io.Reader, psize int, lookahead int) *RtpBridge {
	return &RtpBridge{
		reader: &RtpReader{
			Reader: reader,
			PacketSize: psize,
		},
		seqnum: -1,
		queue: util.NewSequenceQueue(lookahead),
		// this allocates a bit too much (we're only interested in the payload),
		// but will work no matter what size the header has
		buffer: bytes.NewBuffer(make([]byte, 0, psize)),
	}
}

func (b *RtpBridge) packetIntoQueue() error {
	p, err := b.reader.ReadRtpPacket()
	if err != nil {
		return err
	}
	if p.PayloadType != RtpPayloadTypeMP2T {
		// we don't accept anything besides MPEG-2 TS packets
		return ErrRtpBridgeInvalidPacketType
	}
	if len(p.Payload) < MpegTsPacketSize {
		// RTP packets need to contain at least one whole MP2TS packet
		return ErrRtpBridgeInvalidPacketType
	}
	seq := int(p.SequenceNumber)
	// check if this is the first packet at start or after a long drought
	var pos int
	if b.queue.Length() == 0 {
		pos = 0
		b.seqnum = seq
	} else {
		// calculate the queue position relative to the last fetched sequence number
		pos = util.AbsSub(b.seqnum, seq)
		// check if we're too far ahead from the head of the queue
		newseq := pos - (b.queue.Length() - 1)
		if newseq > 0 {
			// to make room for a packet this far ahead, we need to get rid of some previous slots first
			for i := 0; i < newseq; i++ {
				// TODO instead of dropping all of them, we could pop all non-empty slots and enqueue the data.
				// but this would require another queue for the reassembled output...
				// or moving from a pull to push design.
				b.queue.Pop()
			}
			logger.Logkv(
				"event", "error",
				"error", "rtp_drop",
				"packets", newseq,
				"message", fmt.Sprintf("Packets lost: %d", newseq),
			)

			// update pos and seqnum with the new base index
			pos -= newseq
			b.seqnum += newseq
		}
	}
	old, err := b.queue.Insert(pos, p)
	if err != nil {
		// should never happen, maybe just panic here?
		return err
	}
	if old != nil {
		logger.Logkv(
			"event", "error",
			"error", "rtp_dup",
			"message", "Packet with duplicate sequence number overwritten",
		)
	}
	return nil
}

func (b *RtpBridge) nextPacket() (*RtpPacket, error) {
	p, err := b.queue.Peek()
	if err != nil {
		return nil, err
	}
	if p != nil {
		// we already fetched the next packet, just update the head now
		b.queue.Pop()
		// this would panic on type mismatch, but the queue only contains RtpPackets and nils are already excluded
		rtp := p.(*RtpPacket)
		return rtp, nil
	}
	// still waiting for the next packet in line
	return nil, nil
}

func (b *RtpBridge) Read(p []byte) (n int, err error) {
	// TODO MPEG2-TS packets may also be 204 bytes long, which includes a 32-bit checksum.
	// This is currently unsupported.

	// the payload may only contain whole MP2TS packets - we don't do partial reassembly
	if b.buffer.Len() < MpegTsPacketSize {
		// partial or empty - clear the buffer
		b.buffer.Reset()
		// try to read the next packet
		err = b.packetIntoQueue()
		if err != nil {
			return 0, err
		}
		// then pop the next packet from the queue (with possible reordering)
		p, err := b.nextPacket()
		if err != nil {
			return 0, err
		}
		if p == nil {
			// no packet available yet, we need to wait for more segments
			return 0, ErrRtpBridgeWaitingForMore
		}
		b.buffer.Write(p.Payload)
	}
	return b.buffer.Read(p)
}

func (b *RtpBridge) Close() error {
	if c, ok := b.reader.Reader.(io.Closer); ok {
		// close the underlying reader if it supports Close()
		return c.Close()
	}
	return nil
}
