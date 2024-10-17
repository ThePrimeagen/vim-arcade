package packet_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

type TestEncoding struct { }

func (t *TestEncoding) Encoding() packet.Encoding {
    return packet.EncodingCustom
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
    ctx, cancel := context.WithCancel(context.Background())

    g := errgroup.Group{}
    g.Go(func() error {
        return framer.Run(ctx, buf)
    })

    count := 0
    for pkt := range framer.Out() {
        fmt.Printf("pkt: \"%s\" expt: \"%s\"\n", string(pkt.Data()), string(testEncoding))
        require.Equal(t, pkt.Data(), testEncoding)
        count++

        if count == 5 {
            cancel()
        }
    }

    cancel()
    require.Equal(t, count, BUF_COUNT)
    require.NoError(t, g.Wait())
}
