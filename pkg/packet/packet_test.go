package packet_test

import (
	"bytes"
	"context"
	"sync"
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
        pkt, err := framer.Pull()
        require.NoError(t, err)
        require.Equal(t, pkt.Data(), testEncoding)
    }

    pkt, err := framer.Pull()
    require.NoError(t, err)
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

    ch := make(chan *packet.Packet, 10)
    ctx, cancel := context.WithCancel(context.Background())

    var err error = nil
    wait := sync.WaitGroup{}
    wait.Add(1)
    go func() {
        err = packet.FrameReader(ctx, buf, ch)
        wait.Done()
    }()

    for range BUF_COUNT {
        pkt := <- ch
        require.Equal(t, pkt.Data(), testEncoding)
    }

    var pkt *packet.Packet = nil
    select {
    case pkt = <- ch:
    default:
    }
    require.Equal(t, pkt, (*packet.Packet)(nil))

    cancel()
    wait.Wait()

    require.NoError(t, err)
}
