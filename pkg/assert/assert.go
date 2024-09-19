package assert

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
)

// TODO using slog for logging
type AssertData interface {
    Dump() string
}
var assertData map[string]AssertData = map[string]AssertData{}
var writer io.Writer

func AddAssertData(key string, value AssertData) {
	assertData[key] = value
}

func RemoveAssertData(key string) {
	delete(assertData, key)
}

func ToWriter(w io.Writer) {
	writer = w
}

func runAssert(msg string, args ...interface{}) {
    slogValues := []interface{}{
        "msg",
        msg,
        "area",
        "Assert",
    }
    slogValues = append(slogValues, args...)

	for k, v := range assertData {
        slogValues = append(slogValues, k, v.Dump())
	}

    fmt.Fprintf(os.Stderr, "ASSERT\n")
    for i := 0; i < len(slogValues); i += 2 {
        fmt.Fprintf(os.Stderr, "   %s=%v\n", slogValues[i], slogValues[i + 1])
    }
    fmt.Fprintln(os.Stderr, string(debug.Stack()))
    os.Exit(1)
}

// TODO Think about passing around a context for debugging purposes
func Assert(truth bool, msg string, data ...any) {
	if !truth {
		runAssert(msg, data...)
	}
}

func NotNil(item any, msg string) {
	if item == nil {
		slog.Error("NotNil#nil encountered")
		runAssert(msg)
	}
}

func Never(msg string, data ...any) {
    Assert(false, msg, data...)
}

func NoError(err error, msg string, data ...any) {
	if err != nil {
		slog.Error("NoError#error encountered", "error", err)
		runAssert(msg, data...)
	}
}


