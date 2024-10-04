package main

import (
	"fmt"
	"os"
)

func main() {
    fmt.Printf("Here you go: %s\n", os.Getenv("DUMMY"))
}

