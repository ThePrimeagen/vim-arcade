package packet

import (
	"fmt"
	"strconv"
	"strings"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

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

