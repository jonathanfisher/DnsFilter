// DnsFilter is intended to be a lightweight DNS "server" that acts very similarly to PiHole. This program will
// receive DNS requests, do a check for blacklist/whitelist and respond accordingly.

package main

import (
	"log"

	"github.com/jonathanfisher/DnsFilter/server"
)

func main() {
	blacklist, err := server.DomainListFromSources([]string{
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
	})
	if err != nil {
		log.Fatalf("failed to load blacklist: %v", err)
	}

	whitelist := server.DomainList{"google.com"}

	resolver := server.NewServerWithFilters(whitelist, blacklist)

	resolver.Listen()
}
