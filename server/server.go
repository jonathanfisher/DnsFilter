package server

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/jonathanfisher/DnsFilter/statistics"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	DefaultDnsPortNumber = 53
	DefaultDnsPacketLength = 512
)

type dnsServer struct {
	conn *net.UDPConn

	domainListMutex sync.RWMutex
	whitelist DomainList
	blacklist DomainList

	stats *statistics.Statistics
}

type DnsResolver interface {
	Listen()
}

// getAnswerForBlockedQuestion returns an Answer for a Question that is intended to be blocked. The returned answer
// will be of the same type as the question (e.g. IPv4 -> IPv4, IPv6 -> IPv6).
func getAnswerForBlockedQuestion(question *dnsmessage.Question) dnsmessage.Resource {
	var body dnsmessage.ResourceBody

	switch question.Type {
	case dnsmessage.TypeA:
		body = &dnsmessage.AResource{A: [4]byte{0, 0, 0, 0}}
		break

	case dnsmessage.TypeAAAA:
		body = &dnsmessage.AAAAResource{AAAA: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	}
	return dnsmessage.Resource{
		Header: dnsmessage.ResourceHeader{
			Name:   question.Name,
			Type:   question.Type,
			Class:  question.Class,
		},
		Body:   body,
	}
}

// getAnswersForQuestions creates and returns a list of answers for each of the DNS Questions that were received.
// The input slices represent the valid and invalid DNS questions, which correspond to the requests that will be
// forwarded on to an upstream DNS server for valid results, and the invalid questions will get their own special
// responses for blocked questions.
// NOTE: This call will block while waiting for communication with the upstream DNS server.
func (s *dnsServer) getAnswersForQuestions(header *dnsmessage.Header, valid, invalid []dnsmessage.Question) ([]dnsmessage.Resource, error) {
	var answers []dnsmessage.Resource

	for _, question := range invalid {
		answers = append(answers, getAnswerForBlockedQuestion(&question))
	}

	// Query upstream for any requests that have been determined not to be blocked. Note that this can fail in cases
	// where a UDP packet is lost. In this case, we want to simply abort, since the client will retry. To accomplish
	// this, we return an error and the client will handle the error.
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

// logResponse takes raw DNS response bytes and the requester's IP address as inputs, and crafts a log message
// that can be passed to our statistics module that will update metrics accordingly. This seems pretty inefficient,
// since we are unpacking a structure that we previously packed, but I couldn't come up with a good solution for
// parsing dnsmessage.ResourceBody without using a dnsmessage.Parser, and those apparently only seem to work from a
// complete byte-representation of a response.
func (s *dnsServer) logResponse(response []byte, remoteIP net.IP) {
	var parser dnsmessage.Parser
	if _, err := parser.Start(response); err != nil {
		log.Printf("failed to parse response: %v", err)
		return
	}

	if err := parser.SkipAllQuestions(); err != nil && err != dnsmessage.ErrSectionDone {
		log.Printf("failed to skip questions: %v", err)
		return
	}

	for {
		header, err := parser.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		} else if err != nil {
			log.Printf("failed to parse answer: %v", err)
			return
		} else if header.Class != dnsmessage.ClassINET {
			log.Printf("invalid class: %v", header.Class)
			if err = parser.SkipAnswer(); err != nil {
				log.Printf("failed to skip answer: %v", err)
				return
			}
			continue
		}

		switch header.Type {
		case dnsmessage.TypeA:
			if r, err := parser.AResource(); err != nil {
				log.Printf("failed to parse AResource: %v", err)
				if err = parser.SkipAnswer(); err != nil {
					log.Printf("failed to skip answer: %v", err)
					return
				}
			} else {
				s.stats.LogEvent(statistics.Event{
					Client:        remoteIP,
					NameRequested: header.Name.String(),
					IPResponse:    r.A[:],
					Timestamp:     time.Now(),
				})
			}
			break

		case dnsmessage.TypeAAAA:
			if r, err := parser.AAAAResource(); err != nil {
				log.Printf("failed to parse AAAAResource: %v", err)
				if err = parser.SkipAnswer(); err != nil {
					log.Printf("failed to skip answer: %v", err)
					return
				}
			} else {
				s.stats.LogEvent(statistics.Event{
					Client:        remoteIP,
					NameRequested: header.Name.String(),
					IPResponse:    r.AAAA[:],
					Timestamp:     time.Now(),
				})
			}
			break

		default:
			if err = parser.SkipAnswer(); err != nil {
				log.Printf("failed to skip answer: %v", err)
				return
			}
			continue
		}

		if err = parser.SkipAnswer(); err == dnsmessage.ErrSectionDone {
			break
		} else if err != nil {
			log.Printf("failed to skip answer: %v", err)
			return
		}
	}
}

// handleReceivedDnsRequest handles raw bytes that represent a DNS request, and filters the requests into valid and
// invalid ones. Based on these results, a response is formulated and sent back to the remoteAddr.
// NOTE: This is intended to be run as a goroutine, and as such, everything in this call should be thread-safe.
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
	log.Printf("Valid: %v [%v], Invalid: %v [%v]", len(valid), valid, len(invalid), invalid)

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

	// Note: as mentioned above, this can fail if a UDP packet gets dropped. In this case, the UDP connection will
	// time out, and pass back an error. We should simply abort this call and let the client retry the DNS request.
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

	// Update the metrics with the information from the response
	s.logResponse(txMsg, remoteAddr.IP)
}

// SetBlacklist sets the current dnsServer's whitelist to the given list. Note that this utilizes the mutex to be
// thread-safe.
func (s *dnsServer) SetWhitelist(list DomainList) {
	s.domainListMutex.Lock()
	defer s.domainListMutex.Unlock()
	s.whitelist = list
}

// SetBlacklist sets the current dnsServer's blacklist to the given list. Note that this utilizes the mutex to be
// thread-safe.
func (s *dnsServer) SetBlacklist(list DomainList) {
	s.domainListMutex.Lock()
	defer s.domainListMutex.Unlock()
	s.blacklist = list
}

// Listen will listen on port 53 for UDP packets, and when they are received, will determine whether they are meant to
// be blocked or whether they need to be passed along to the upstream nameserver.
// NOTE: This call will block indefinitely.
func (s dnsServer) Listen() {
	var err error

	s.conn, err = net.ListenUDP("udp", &net.UDPAddr{Port: DefaultDnsPortNumber})
	if err != nil {
		log.Fatalf("Failed to listen to UDP port %v: %v", DefaultDnsPortNumber, err)
	}
	defer s.conn.Close()

	log.Printf("Listening on port %v", DefaultDnsPortNumber)
	for {
		packetBuffer := make([]byte, DefaultDnsPacketLength)
		_, remoteAddr, err := s.conn.ReadFromUDP(packetBuffer)
		if err != nil {
			log.Fatalf("Failed to read packets from UDP port: %v", err)
		}

		go s.handleReceivedDnsRequest(packetBuffer, remoteAddr)
	}
}

// NewServer creates and returns a default DnsResolver
func NewServer() DnsResolver {
	return dnsServer{stats: statistics.New()}
}

// NewServerWithFilters creates and returns a DnsResolver with the given whitelist and blacklist in place.
func NewServerWithFilters(whitelist, blacklist DomainList) DnsResolver {
	return dnsServer{
		whitelist: whitelist,
		blacklist: blacklist,
		stats:     statistics.New(),
	}
}