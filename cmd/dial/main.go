package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type SpoofIp struct {
    Port int
    Source string
}

//Network() string // name of the network (for example, "tcp", "udp")
//String() string  // string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
func (s *SpoofIp) String() string {
    return fmt.Sprintf("%s:%d", s.Source, s.Port)
}

func (s *SpoofIp) Network() string {
    return "tcp"
}

func main() {
    var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

    addr := net.TCPAddr{IP: net.IPv4(127, 0, 69, 69), Port: 42042}
    d.LocalAddr = &addr

	conn, err := d.DialContext(ctx, "tcp", "127.0.42.69:42069")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("Hello, World!")); err != nil {
		log.Fatal(err)
	}
}
