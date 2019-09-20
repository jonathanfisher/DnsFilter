// This file contains the logic necessary for sending requests upstream, and receiving responses to those requests.
package server

import (
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

var DefaultDnsServerList = []net.IP {
	net.ParseIP("8.8.8.8"),
	net.ParseIP("8.8.4.4"),
	net.ParseIP("2001:4860:4860::8888"),
	net.ParseIP("2001:4860:4860::8844"),
}

func getDnsServer() net.IP {
	return DefaultDnsServerList[0]
}

// QueryUpstreamDns takes a pre-constructed DNS request message and sends it to an upstream DNS server. Then wait for
// a response from that server and return it to the caller.
// TODO: An obvious optimization here would be to have one goroutine that handles all traffic between here & upstream.
// TODO: What do we do if we don't receive a response in a reasonable amount of time (i.e. when a packet is lost)
func (s *dnsServer) QueryUpstreamDns(message *dnsmessage.Message) (dnsmessage.Message, error) {
	packedMessage, err := message.Pack()
	if err != nil {
		return dnsmessage.Message{}, err
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   nil, 	// Listen on all interfaces
		Port: 0, 	// Select a port automatically
	})
	if err != nil {
		return dnsmessage.Message{}, err
	}
	defer conn.Close()

	_, err = conn.WriteToUDP(packedMessage, &net.UDPAddr{
		IP:   getDnsServer(),
		Port: DefaultDnsPortNumber,
	})
	if err != nil {
		return dnsmessage.Message{}, err
	}

	// Wait for the response to come back from upstream
	rxBuf := make([]byte, DefaultDnsPacketLength)
	_, _, err = conn.ReadFromUDP(rxBuf)
	if err != nil {
		return dnsmessage.Message{}, err
	}

	var rxMsg dnsmessage.Message
	err = rxMsg.Unpack(rxBuf)
	if err != nil {
		return dnsmessage.Message{}, err
	}
	return rxMsg, nil
}