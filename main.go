// DnsFilter is intended to be a lightweight DNS "server" that acts very similarly to PiHole. This program will
// receive DNS requests, do a check for blacklist/whitelist and respond accordingly.

package main

import (
	"log"

	"github.com/jonathanfisher/DnsFilter/server"
)

func main() {
	resolver, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	resolver.Listen()
}
