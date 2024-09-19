package servermanagement

import "errors"

var NO_BEST_SERVER = errors.New("no best server found")

type ServerParams struct {
    MaxConnections int

    // TODO can i do this?
    MaxLoad float32
}
