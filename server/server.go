package server

import (
	"log"
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	DefaultDnsPortNumber = 53
	DefaultDnsPacketLength = 512
)

type dnsServer struct {
	conn *net.UDPConn
}

type DnsResolver interface {
	Listen()
}

func getAnswerForBlockedQuestion(question *dnsmessage.Question) dnsmessage.Resource {
	return dnsmessage.Resource{
		Header: dnsmessage.ResourceHeader{
			Name:   question.Name,
			Type:   question.Type,
			Class:  question.Class,
		},
		Body:   &dnsmessage.AResource{A: [4]byte{0, 0, 0, 0}},
	}
}

func (s *dnsServer) getAnswersForQuestions(header *dnsmessage.Header, valid, invalid []dnsmessage.Question) ([]dnsmessage.Resource, error) {
	var answers []dnsmessage.Resource

	for _, question := range invalid {
		answers = append(answers, getAnswerForBlockedQuestion(&question))
	}

	upstreamResponse, err := s.QueryUpstreamDns(&dnsmessage.Message{
		Header:      *header,
		Questions:   valid,
		Answers:     nil,
		Authorities: nil,
		Additionals: nil,
	})
	if err != nil {
		return nil, err
	}

	answers = append(answers, upstreamResponse.Answers...)

	return answers, nil
}

func (s *dnsServer) handleReceivedDnsRequest(buf []byte, remoteAddr *net.UDPAddr) {
	var msg dnsmessage.Message

	if err := msg.Unpack(buf); err != nil {
		log.Printf("Failed to parse DNS Request: %v", err)
		return
	}

	// Make sure there is at least one question in the request.
	if len(msg.Questions) == 0 {
		log.Printf("received request with no questions, ignoring")
		return
	}

	valid, invalid := s.Filter(&msg)
	log.Printf("Valid: %v, Invalid: %v", len(valid), len(invalid))

	// Create the response that we will use.
	dnsResponse := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:            msg.ID,
			Response:      true,
			Authoritative: true,
		},
		Questions:   nil,
		Answers:     nil,
		Authorities: nil,
		Additionals: nil,
	}
	answers, err := s.getAnswersForQuestions(&msg.Header, valid, invalid)
	if err != nil {
		log.Printf("failed to get answers: %v", err)
		return
	}

	dnsResponse.Answers = answers

	// Now that we have a response, send it back to the caller
	txMsg, err := dnsResponse.Pack()
	if err != nil {
		log.Printf("failed to pack response: %v", err)
	}

	_, err = s.conn.WriteToUDP(txMsg, remoteAddr)
	if err != nil {
		log.Printf("failed to write response to client: %v", err)
	}
}

func (s dnsServer) Listen() {
	var err error

	s.conn, err = net.ListenUDP("udp", &net.UDPAddr{Port: DefaultDnsPortNumber})
	if err != nil {
		log.Fatalf("Failed to listen to UDP port %v: %v", DefaultDnsPortNumber, err)
	}
	defer s.conn.Close()

	for {
		packetBuffer := make([]byte, DefaultDnsPacketLength)
		_, remoteAddr, err := s.conn.ReadFromUDP(packetBuffer)
		if err != nil {
			log.Fatalf("Failed to read packets from UDP port: %v", err)
		}

		go s.handleReceivedDnsRequest(packetBuffer, remoteAddr)
	}
}

// NewServer creates and returns a newly created DnsResolver
func NewServer() (DnsResolver, error) {
	var server dnsServer

	return server, nil
}