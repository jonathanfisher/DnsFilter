package server

import (
	"golang.org/x/net/dns/dnsmessage"
	"strings"
)

type DnsFilter interface {
	Filter(*dnsmessage.Message) ([]dnsmessage.Question, []dnsmessage.Question)
}

func (s *dnsServer) domainIsWhitelisted(domain string) bool {
	s.domainListMutex.RLock()
	defer s.domainListMutex.RUnlock()
	return s.whitelist.Contains(domain)
}

func (s *dnsServer) domainIsBlacklisted(domain string) bool {
	s.domainListMutex.RLock()
	defer s.domainListMutex.RUnlock()
	return s.blacklist.Contains(domain)
}

// Filter iterates through a DNS Request message's Questions and sorts them into valid and invalid lists.
// The logic is fairly straightforward: if a question is whitelisted or not blacklisted, we forward it on
// to the upstream DNS server, otherwise we mark it as invalid and return a result of 127.0.0.1.
func (s *dnsServer) Filter(msg *dnsmessage.Message) ([]dnsmessage.Question, []dnsmessage.Question) {
	var valid, invalid []dnsmessage.Question

	for _, question := range msg.Questions {
		// If the domain contains a trailing period, we trim it here.
		domain := strings.TrimSuffix(question.Name.String(), ".")

		if s.domainIsWhitelisted(domain) || !s.domainIsBlacklisted(domain) {
			valid = append(valid, question)
		} else {
			invalid = append(invalid, question)
		}
	}

	return valid, invalid
}