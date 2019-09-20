package server

import "golang.org/x/net/dns/dnsmessage"

type DnsFilter interface {
	Filter(*dnsmessage.Message) ([]dnsmessage.Question, []dnsmessage.Question)
}

func domainIsWhitelisted(domain string) bool {
	return false
}

func domainIsBlacklisted(domain string) bool {
	return false
}

// Filter iterates through a DNS Request message's Questions and sorts them into valid and invalid lists.
// The logic is fairly straightforward: if a question is whitelisted or not blacklisted, we forward it on
// to the upstream DNS server, otherwise we mark it as invalid and return a result of 127.0.0.1.
func (s *dnsServer) Filter(msg *dnsmessage.Message) ([]dnsmessage.Question, []dnsmessage.Question) {
	var valid, invalid []dnsmessage.Question

	for _, question := range msg.Questions {
		if domainIsWhitelisted(question.Name.GoString()) || !domainIsBlacklisted(question.Name.GoString()) {
			valid = append(valid, question)
		} else {
			invalid = append(invalid, question)
		}
	}

	return valid, invalid
}