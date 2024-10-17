package packet_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

type TestEncoding struct { }

func (t *TestEncoding) Encoding() packet.Encoding {
    return packet.EncodingString
}

func (t *TestEncoding) Type() uint8 {
    return 63 // doesn't really matter
}

var testEncoding = []byte("hello encoding")
func (t *TestEncoding) Read(data []byte) (int, error) {
    return copy(data, testEncoding), nil
}

func TestPacketCreation(t *testing.T) {
    te := TestEncoding{}
    p := packet.NewPacket(&te)

    data := make([]byte, 0, 100)
    buf := bytes.NewBuffer(data)

    n, err := p.Into(buf)

    require.NoError(t, err, "into had an error")
    require.Equal(t, n, packet.HEADER_SIZE + len(testEncoding), "expected n to have the same value as len of testEncoding")
    require.Equal(t, testEncoding, data[packet.HEADER_SIZE:n], "expected encoding to have the same value as testEncoding")
    require.Equal(t, testEncoding, p.Data(), "expected encoding to have the same value as testEncoding")
}

func TestPacketFramer(t *testing.T) {
    te := TestEncoding{}
    p := packet.NewPacket(&te)

    data := make([]byte, 0, 100)
    buf := bytes.NewBuffer(data)

    BUF_COUNT := 5

    for range BUF_COUNT {
        _, err := p.Into(buf)
        require.NoError(t, err, "unable to write into buffer")
    }

    framer := packet.NewPacketFramer()
    framer.Push(buf.Bytes())

    for range BUF_COUNT {
        pkt := <-framer.C
        require.Equal(t, pkt.Data(), testEncoding)
    }

    var pkt *packet.Packet
    select {
    case p := <-framer.C:
        pkt = p
    default:
    }

    require.Equal(t, pkt, (*packet.Packet)(nil))
}

func TestReaderFramer(t *testing.T) {
    te := TestEncoding{}
    p := packet.NewPacket(&te)

    data := make([]byte, 0, 100)
    buf := bytes.NewBuffer(data)

    BUF_COUNT := 5

    for range BUF_COUNT {
        _, err := p.Into(buf)
        require.NoError(t, err, "unable to write into buffer")
    }

    framer := packet.NewPacketFramer()

    var err error = nil
    go func() {
        err = packet.FrameWithReader(&framer, buf)
        if !errors.Is(err, io.EOF) {
            require.NoError(t, err)
        }
    }()

    for range BUF_COUNT {
        pkt := <- framer.C
        require.Equal(t, pkt.Data(), testEncoding)
    }

    var pkt *packet.Packet = nil
    select {
    case pkt = <- framer.C:
    default:
    }
    require.Equal(t, pkt, (*packet.Packet)(nil))
    if !errors.Is(err, io.EOF) {
        require.NoError(t, err)
    }
}

func TestPacketFromParts(t *testing.T) {
    p := packet.CreateClientAuth([]byte{
        0, 4, 2, 0,
        1, 3, 3, 7,
        0, 0, 4, 2,
        0, 0, 6, 9,
    })

    data := make([]byte, 0, 100)
    buf := bytes.NewBuffer(data)

    var err error = nil
    _, err = p.Into(buf)
    require.NoError(t, err, "unable to write into buffer")

    pkt := buf.Bytes()
    pktFromBytes := packet.PacketFromBytes(pkt)
    bLen := binary.BigEndian.Uint16(pkt[2:])

    require.Equal(t, pktFromBytes, p)
    require.Equal(t, pkt[0], packet.VERSION)
    require.Equal(t, pkt[1], packet.CreateTypeAndEncodingByte(packet.PacketClientAuth, packet.EncodingBytes))
    require.Equal(t, bLen, uint16(16))
}
