package utils

import (
	"testing"
)

type buggyWriter struct {
	buf []byte
}

func (b *buggyWriter) Write(p []byte) (n int, err error) {
	// This Writer writes at most 10 bytes to its internal buffer.
	// This implementation is intentionally not conforming to the io.Writer spec,
	// which states "Write must return a non-nil error if it returns n < len(p)."
	l := min(len(p), 10)

	b.buf = append(b.buf, p[:l]...)
	return l, nil
}

func TestWriteAllWithBuggyWriter(t *testing.T) {
	data := []byte{
		'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a',
		'b', 'b', 'b', 'b', 'b', 'b', 'b', 'b', 'b', 'b',
		'c', 'c', 'c', 'c', 'c', 'c', 'c', 'c', 'c', 'c',
	}

	w := &buggyWriter{}

	err := WriteAll(data, w)
	if err != nil {
		t.Fatalf("Write failed with an error: %s", err)
	}

	if string(data) != string(w.buf) {
		t.Logf("Buffer contents did not match: expected=%q, actual=%q", data, w.buf)
		t.Fail()
	}
}
