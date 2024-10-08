package packet

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

type PacketType int

const (
    ClientHello PacketType = iota
    ClientClosed

    MatchMakingRedirect
)

type Packet struct {
    Type PacketType
    Data []byte
}

func NewPacket(data []byte) Packet {
    // I can return the type and everything here..
    return Packet {
        Data: data,
        Type: ClientClosed,
    }
}

// this could be a lot better...
type PacketParser struct {
    buf []byte
    in <-chan Packet
    out chan<- Packet
    errIn <-chan error
    errOut chan<- error
}

func NewPacketParser() PacketParser {
    ch := make(chan Packet, 10)
    err := make(chan error, 10)

    return PacketParser{
        buf: []byte{},
        in: ch,
        out: ch,
        errIn: err,
        errOut: err,
    }
}

var newLine = []byte("\n")
func (p *PacketParser) Push(data []byte) {
    p.buf = append(p.buf, data...)

    for {
        idx := bytes.Index(p.buf, newLine)
        if idx == -1 {
            break
        }

        buf := p.buf[0:idx]
        p.buf = p.buf[idx + 1:]
        p.out <- NewPacket(buf)
    }
}

func (p *PacketParser) OnError() <-chan error {
    assert.NotNil(p.errIn, "OnError has been called already")

    ch := p.errIn
    p.errIn = nil

    return ch
}

func (p *PacketParser) OnData() <-chan Packet {
    assert.NotNil(p.in, "OnData has been called already")

    ch := p.in
    p.in = nil

    return ch
}

func MakeClientHello(id string) Packet {
    return Packet {
        Type: ClientHello,
        Data: []byte(fmt.Sprintf("client:%s\n", id)),
    }
}

func MakeClientClose() Packet {
    return Packet {
        Type: ClientClosed,
        Data: []byte("close\n"),
    }
}

func IsEmpty(data []byte) bool {
    return len(data) == 0
}

func IsClientClosed(data []byte) bool {
    return string(data) == "close"
}

func LegacyClientId(id string) []byte {
    return []byte(fmt.Sprintf("hello:%s", id))
}

func ParseClientId(packet string) (int, error) {
    assert.Assert(strings.HasPrefix(packet, "hello:"), "passed in a non hello packet to ParseClientId", "packet", packet)
    return strconv.Atoi(strings.TrimSpace(packet[6:]))
}

func LegacyClientClose() []byte {
    return []byte("close")
}

func LegacyIsEmpty(data []byte) bool {
    return len(data) == 0
}

func LegacyIsClientClosed(data []byte) bool {
    return string(data) == "close"
}

