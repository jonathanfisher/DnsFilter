// DnsFilter is intended to be a lightweight DNS "server" that acts very similarly to PiHole. This program will
// receive DNS requests, do a check for blacklist/whitelist and respond accordingly.

package main

import (
	"log"

	"github.com/jonathanfisher/DnsFilter/hosts"
	"github.com/jonathanfisher/DnsFilter/server"
)

func main() {
	var err error
	
	_, err = hosts.ParseUrl("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
	if err != nil {
		panic(err)
	}

	resolver, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	resolver.Listen()
}
