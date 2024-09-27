package utils

import (
	"context"
	"errors"
	"io"
)

const DateTimeFormatForSQLite = "2006-01-02 15:04:05"

type ContextReader struct {
    Err chan error
    Out chan []byte
    ctx context.Context
}

func NewContextReader(ctx context.Context) ContextReader {
    return ContextReader{
        Err: make(chan error, 1),
        Out: make(chan []byte, 10),
        ctx: ctx,
    }
}

func internalRead(in io.Reader, out chan []byte, err chan error) {
    data := make([]byte, 1024, 1024)
    for {
        n, e := in.Read(data)
        if e != nil {
            if !errors.Is(e, io.EOF) {
                err <- e
            }
            break
        }

        o := make([]byte, n, n)
        copy(o, data[0:n])

        out <- o
    }
}

func (c *ContextReader) Read(in io.Reader) {
    go func() {
        ctx, cancel := context.WithCancel(c.ctx)
        defer func() {
            close(c.Err)
            close(c.Out)
        }()

        err := make(chan error, 1)
        go internalRead(in, c.Out, err)
        go func() {
            e := <-err
            c.Err <- e
            cancel()
        }()
        <-ctx.Done()
    }()
}

