package packet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/utils"
)

const VERSION = 1
const HEADER_SIZE = 4
const HEADER_LENGTH_OFFSET = 2
const PACKET_MAX_SIZE = 1024
const PACKET_PAYLOAD_SIZE = 1024 - HEADER_SIZE

var PacketMaxSizeExceeded = errors.New(fmt.Sprintf("Packet length has exceeded allowed size of %d", PACKET_PAYLOAD_SIZE - 1))

type Packet struct {
    data []byte
    len int
}

type PacketEncoder interface {
    io.Reader
    Type() uint8
    Encoding() uint8
}

func NewPacket(encoder PacketEncoder) Packet {
    b := make([]byte, PACKET_MAX_SIZE, PACKET_MAX_SIZE)

    enc := b[HEADER_SIZE:]
    n, err := encoder.Read(enc)
    assert.NoError(err, "i should never fail on encoding a packet")
    assert.Assert(n != PACKET_PAYLOAD_SIZE, "max packet size exceeded", "MAX_SIZE", PACKET_PAYLOAD_SIZE)

    b[0] = VERSION
    b[1] = encoder.Encoding() << 7 | encoder.Type()

    binary.BigEndian.PutUint16(b[2:], uint16(n))

    return Packet{data: b, len: n}
}

func (p *Packet) Into(writer io.Writer) error {
    // TODO v2
    // reconsider just trusting the science and if it fails, assert and report
    // to golang's github
    return utils.WriteAll(p.data[:p.len], writer)
}

type PacketFramer struct {
    buf []byte
    idx int
    out chan []byte
}

func NewPacketFramer() PacketFramer {
    return PacketFramer{
        buf: make([]byte, PACKET_PAYLOAD_SIZE, PACKET_PAYLOAD_SIZE),
        out: make(chan []byte, 10),
    }
}

func (p *PacketFramer) read() error {
    if len(p.buf) > HEADER_SIZE {
        packetLen := binary.BigEndian.Uint16(p.buf[HEADER_LENGTH_OFFSET:])
        if packetLen == PACKET_PAYLOAD_SIZE {
            return PacketMaxSizeExceeded
        }

        if packetLen + HEADER_SIZE <= uint16(p.idx) {
            out := make([]byte, packetLen, packetLen)
            copy(out, p.buf[:packetLen])
            copy(p.buf, p.buf[packetLen + HEADER_SIZE:])
            p.idx = p.idx - (int(packetLen) + HEADER_SIZE)

            p.out <- out
        }
    }

    return nil
}

func (p *PacketFramer) Run(ctx context.Context, r io.Reader) error {
    reader := utils.NewContextReader(ctx)
    reader.Read(r)

    outer:
    for {
        select {
        case d := <-reader.Out:
            read := 0
            for read < len(d) {
                n := copy(p.buf[p.idx:], d[read:])
                p.idx += n
                read += n
                p.read()
            }

        case e := <-reader.Err:
            return e

        case <-ctx.Done():
            break outer
        }
    }

    return nil
}






