package server

import (
	"log"

	"github.com/jonathanfisher/DnsFilter/hosts"
)

type DomainList []string

// Contains iterates through all items in the DomainList slice and returns true if the given string is already there,
// otherwise false.
func (d DomainList) Contains(domain string) bool {
	for _, v := range d {
		if v == domain {
			return true
		}
	}

	return false
}

// DomainListFromSources builds a list of domains from a list of sources. Those sources are expected to be in
// Host file format. This *should* work with both local files and remote ones, but for now only URLs have been
// tested. The hosts file is parsed, but then we discard all resolution information (i.e. we throw away whatever
// the upstream-given resolution is for a given name).
func DomainListFromSources(sources []string) (DomainList, error) {
	var list DomainList

	for _, source := range sources {
		if m, err := hosts.ParseUrl(source); err != nil {
			log.Printf("failed to load source %v: %v", source, err)
			return nil, err
		} else {
			for k := range m {
				if list == nil || !list.Contains(k) {
					list = append(list, k)
				}
			}
		}
	}

	return list, nil
}

// Union creates and returns the union of two DomainLists. Note: this function call assumes that both input DomainLists
// are duplicate free (e.g. "a" has no duplicate entries and "b" has no duplicate entries).
func Union(a, b DomainList) DomainList {
	var combined DomainList

	for _, domain := range a {
		combined = append(combined, domain)
	}

	for _, domain := range b {
		if combined == nil || !combined.Contains(domain) {
			combined = append(combined, domain)
		}
	}

	return combined
}
