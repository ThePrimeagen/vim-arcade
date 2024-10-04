package servermanagement

import "errors"

var NoBestServer = errors.New("no best server found")

type ServerParams struct {
    MaxLoad float32
}
