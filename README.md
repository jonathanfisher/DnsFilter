# DnsFilter
This project is intended to be roughly functionally similar to [Pi-hole](https://pi-hole.net/). I make no guarantees
for how well it works, what bugs it may have, etc.

## Overview
This program is intended to run as a DNS server on a local network. It will receive packets on the local UDP port 53,
and will try to determine whether or not to pass that request along to an upstream server. Blocked requests will be
answered with a result of `0.0.0.0`. There is currently no caching done; that should be added eventually.

## Blacklist
The blacklist is currently done as a union of Hosts-formatted files that the Pi-hole community has developed. Right
now there is no great way to configure the app at runtime, so you'll have to make a source file change. In `main.go`,
you'll see stubs that indicate how to build your blacklist (and whitelist, for good measure).

Regular expressions are not supported at this time. Eventually they could be added, but for now it was simpler to
filter based only on explicit matches.