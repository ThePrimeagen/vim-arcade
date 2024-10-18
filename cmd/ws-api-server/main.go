package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {

    // TODO create a webserver mechanism/env for websockets instead of tcp
	portStr := fmt.Sprintf(":80")
	listener, err := net.Listen("tcp", portStr)
    if err != nil {
        fmt.Printf("OH NO LISTENER!!!: %s\n", err.Error())
        return
    }

    for {
        c, err := listener.Accept()
        if err != nil {
            fmt.Printf("OH NO CONNECTION ERROR: %s\n", err.Error())
            continue
        }

        // TODO to move to cloudflare i will need to use websockets for
        //
        // client -> game proxy server
        //
        // tcp and the packets can be used within the ecosystem (auth server ->
        // games), but not without (client -> cloudflare).  so.. kind of sucks
        // to have a double wrapper and is quite inefficient but to have
        // Cloudflare Spectrum (specifically stating support for gaming) you
        // have to have enterprise plan.  I personally would like to avoid the
        // Trust & Safety team, i.e.: Sales team, as long as possible, so i
        // will just use websockets
        c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
    }

}
