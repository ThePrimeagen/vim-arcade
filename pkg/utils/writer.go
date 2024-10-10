package utils

import (
	"io"
)

func WriteAll(data []byte, writer io.Writer) error {
    wrote := 0
    for {
        if wrote == len(data) {
            break
        }

        n, err := writer.Write(data)
        if err != nil {
            return err
        }

        wrote += n
    }

    return nil
}

