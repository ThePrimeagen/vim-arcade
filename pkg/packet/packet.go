package packet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"vim-arcade.theprimeagen.com/pkg/assert"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
	"vim-arcade.theprimeagen.com/pkg/utils"
)

const VERSION uint8 = 1

const HEADER_SIZE = 4
const TYPE_ENC_INDEX = 1
const MAX_TYPE_SIZE = 0x3F
const HEADER_LENGTH_OFFSET = 2
const PACKET_MAX_SIZE = 1024
const PACKET_PAYLOAD_SIZE = 1024 - HEADER_SIZE

const PACKET_AUTH_SIZE = 16 + HEADER_SIZE

var PacketMaxSizeExceeded = fmt.Errorf("Packet length has exceeded allowed size of %d", PACKET_PAYLOAD_SIZE - 1)
var PacketVersionMismatch = fmt.Errorf("Expected packet version to equal %d", VERSION)
var PacketBufferNotBigEnough = fmt.Errorf("Buffer could not fit the entire packet")

type Encoding uint8

const (
    EncodingJSON Encoding = iota
    EncodingString
    EncodingBytes
    EncodingUNUSED2
)

type PacketType uint8

const (
    PacketError PacketType = iota
    PacketMessage
    PacketClientAuth
    PacketServerAuthResponse
    PacketGameSettings
    PacketItem
    PacketItemUpdate
    PacketCloseConnection
)

type Packet struct {
    data []byte
    len int
}

type PacketEncoder interface {
    io.Reader
    Type() uint8
    Encoding() Encoding
}

func TypeToString(t PacketType) string {
    // TODO could be a sweet short :)
    switch t {
    case PacketError: return "Error"
    case PacketMessage: return "Message"
    case PacketClientAuth: return "ClientAuth"
    case PacketServerAuthResponse: return "ServerAuthResponse"
    case PacketGameSettings: return "GameSettings"
    case PacketItem: return "Item"
    case PacketItemUpdate: return "ItemUpdate"
    case PacketCloseConnection: return "CloseConnection"
    default:
        assert.Never("packet unknown", "type", t)
    }
    return ""
}

func CreateTypeAndEncodingByte(t PacketType, enc Encoding) byte {
    return uint8(enc << 6) | uint8(t)
}

func PacketFromParts(t PacketType, enc Encoding, data []byte) Packet {
    assert.Assert(len(data) < PACKET_PAYLOAD_SIZE, "packet size is too large", "MAX", PACKET_MAX_SIZE - 1, "received", len(data))
    assert.Assert(t < MAX_TYPE_SIZE, "max type size exceeded", "MAX", MAX_TYPE_SIZE - 1, "received", t)

    buf := append([]byte{
        VERSION,
        CreateTypeAndEncodingByte(t, enc),
        0,
        0,
    }, data...)

    binary.BigEndian.PutUint16(buf[HEADER_LENGTH_OFFSET:], uint16(len(data)))

    return Packet{
        data: buf,
        len: len(buf),
    }
}

func CreateMessage(msg string) Packet {
    return PacketFromParts(PacketMessage, EncodingString, []byte(msg))
}

func CreateErrorPacket(err error) Packet {
    return PacketFromParts(PacketError, EncodingString, []byte(err.Error()))
}

func CreateServerAuthResponse(accepted bool, id string) Packet {
    var b uint8 = 1
    if !accepted {
        b = 0
    }

    data := []byte{ b }
    data = append(data, []byte(id)...)

    return PacketFromParts(PacketServerAuthResponse, EncodingBytes, data)
}

func CreateCloseConnection() Packet {
    // i think i have a 0 packet size assert...
    // lets find out
    return PacketFromParts(PacketCloseConnection, EncodingBytes, []byte{})
}

func CreateClientAuth(id []byte) Packet {
    assert.Assert(len(id) == 16, "cannot create a auth packet that isn't 16 bytes", "len", len(id))
    return PacketFromParts(PacketClientAuth, EncodingBytes, id)
}

func getPacketLength(data []byte) uint16 {
    return binary.BigEndian.Uint16(data[HEADER_LENGTH_OFFSET:])
}

func PacketFromBytes(data []byte) Packet {
    assert.Assert(data[0] == VERSION, "version mismatch: this should be handled by the framer before packet is created", "VERSION", VERSION, "provided", data[0])

    dataLen := len(data) - HEADER_SIZE
    assert.Assert(dataLen >= 0, "packets must contain some sort of data")

    encodedLen := getPacketLength(data)
    assert.Assert(dataLen == int(encodedLen), "the data buffer provided has a length mismatch", "expected length", dataLen, "encoded length", encodedLen)

    return Packet{
        data: data,
        len: len(data),
    }
}

func NewPacket(encoder PacketEncoder) Packet {
    b := make([]byte, PACKET_MAX_SIZE, PACKET_MAX_SIZE)

    enc := b[HEADER_SIZE:]
    n, err := encoder.Read(enc)
    assert.NoError(err, "i should never fail on encoding a packet")
    assert.Assert(n != PACKET_PAYLOAD_SIZE, "max packet size exceeded", "MAX_SIZE", PACKET_PAYLOAD_SIZE)

    t := encoder.Type()
    assert.Assert(t <= (0x40 - 1), "type has exceeded allowed size", "type", t)

    b[0] = VERSION
    b[1] = uint8(encoder.Encoding() << 6) | encoder.Type()

    binary.BigEndian.PutUint16(b[2:], uint16(n))

    return Packet{data: b, len: n + HEADER_SIZE}
}

func (p *Packet) Into(writer io.Writer) (int, error) {
    return writer.Write(p.data[:p.len])
}

func (p *Packet) Len() uint16 {
    return binary.BigEndian.Uint16(p.data[2:])
}

func (p *Packet) Data() []byte {
    return p.data[HEADER_SIZE:p.len]
}

func (p *Packet) Type() PacketType {
    return PacketType(p.data[TYPE_ENC_INDEX] & 0x3F)
}

func (p *Packet) Encoding() Encoding {
    return Encoding((p.data[TYPE_ENC_INDEX] >> 6) & 0x3)
}

func (p *Packet) Read(data []byte) (int, error) {
    if len(data) < p.len {
        return 0, PacketBufferNotBigEnough
    }
    copy(data, p.data[0:p.len])
    return p.len, nil
}

func (p *Packet) String() string {
    prettyData := utils.PrettyPrintBytes(p.Data(), 16)
    return fmt.Sprintf("Packet(v=%d, t=%s, enc=%d, len=%d) -> \"%s\"", p.data[0], TypeToString(p.Type()), p.Encoding(), p.Len(), prettyData)
}

type PacketFramer struct {
    buf []byte
    idx int
    C chan *Packet
}

func NewPacketFramer() PacketFramer {
    return PacketFramer{
        buf: make([]byte, PACKET_PAYLOAD_SIZE, PACKET_PAYLOAD_SIZE),
        C: make(chan *Packet, 10),
    }
}

func (p *PacketFramer) Push(data []byte) error {
    n := copy(p.buf[p.idx:], data)

    if n < len(data) {
        p.buf = append(p.buf, data[n:]...)
    }

    p.idx += len(data)

    prettylog.Trace(slog.Default(),"PacketFramer received bytes", "len", p.idx, "pretty bytes", utils.PrettyPrintBytes(p.buf, p.idx))

    for {
        pkt, err := p.pull()
        if err != nil || pkt == nil {
            return err
        }

        p.C <- pkt
    }
}

func (p *PacketFramer) pull() (*Packet, error) {
    if p.idx < HEADER_SIZE {
        return nil, nil
    }

    if p.buf[0] != VERSION {
        return nil, errors.Join(
            PacketVersionMismatch,
            fmt.Errorf("received version: %d", p.buf[0]))
    }

    packetLen := getPacketLength(p.buf)
    fullLen := packetLen + HEADER_SIZE
    if packetLen == PACKET_PAYLOAD_SIZE {
        return nil, PacketMaxSizeExceeded
    }

    if fullLen <= uint16(p.idx) {
        out := make([]byte, fullLen, fullLen)
        copy(out, p.buf[:fullLen])
        copy(p.buf, p.buf[fullLen:])
        p.idx = p.idx - int(fullLen)

        pkt := PacketFromBytes(out)
        return &pkt, nil
    }

    return nil, nil
}

func FrameWithReader(framer *PacketFramer, reader io.Reader) error {
    data := make([]byte, 100, 100)
    for {
        n, err := reader.Read(data)
        if err != nil {
            return err
        }

        framer.Push(data[:n])
    }
}

func IsCloseConnection(p *Packet) bool {
    return p.Type() == PacketCloseConnection
}
func IsServerAuth(p *Packet) bool {
    return p.Type() == PacketServerAuthResponse
}

func ServerAuthGameId(p *Packet) string {
    assert.Assert(p.Type() == PacketServerAuthResponse, "cannot cast the packet into a server auth packet", "packet", p.String())
    return string(p.data[HEADER_SIZE + 1:])
}
