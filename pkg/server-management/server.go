package servermanagement

import "errors"

var NoBestServer = errors.New("no best server found")

type ServerParams struct {
    MaxConnections int

    // TODO can i do this?
    MaxLoad float32
}
