package amproxy

import (
	"os"
	"strconv"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

type AMProxyConfig struct {
    AuthTimeoutMS int64 `json:"authTimeoutMS"`
}

func readInt(key string, d int) int {
    vStr := os.Getenv(key)
    v, err := strconv.Atoi(vStr)
    assert.Assert(err == nil || err != nil && len(vStr) == 0, "environment provided an invalid int")

    if err != nil {
        return d
    }

    return v
}

func AMProxyConfigFromEnv() AMProxyConfig {
    return AMProxyConfig{
        AuthTimeoutMS: int64(readInt("AUTH_TIMEOUT_MS", 5000)),
    }
}

