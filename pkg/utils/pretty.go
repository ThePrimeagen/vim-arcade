package utils

import (
	"fmt"
	"strings"
)

func PrettyPrintBytes(data []byte, printMax int) string {
    out := []string{}
    printOut := min(len(data), 16)
    ellipse := len(data) > printOut

    for i := range printOut {
        out = append(out, fmt.Sprintf("%02x", data[i]))
    }

    if ellipse {
        out = append(out, "...")
    }

    return strings.Join(out, " ")
}

