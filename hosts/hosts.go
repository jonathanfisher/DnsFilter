package hosts

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type Map map[string]net.IP

func clearComments(s string) string {
	parts := strings.Split(s, "#")
	return parts[0]
}

func parseIP(ip string) net.IP {
	// Strip out the interface information (if it exists).
	// TODO: Figure out how to use this information.
	return net.ParseIP(strings.Split(ip, "%")[0])
}

// Parse takes an input stream and attempts to parse out the hosts file map. If it fails, an error is returned.
func Parse(input io.ReadCloser) (Map, error) {
	scanner := bufio.NewScanner(input)

	var hostsMap Map
	hostsMap = make(Map)

	for scanner.Scan() {
		s := scanner.Text()
		withoutComments := clearComments(s)
		if len(withoutComments) > 0 {
			// Parse the hosts file format. The format is whitespace-separated fields, starting with the IP
			// address that the following fields should resolve to.
			fields := strings.Fields(withoutComments)
			if len(fields) >= 2 {
				// Parse the IP from the first field. Note that this field can optionally contain interface information
				// For example: fe80::1%lo0 indicates using the lo0 interface. Right now, this information is being
				// discarded.
				// TODO: Figure out how to incorporate this information.
				ip := parseIP(fields[0])

				if ip == nil {
					log.Printf("Invalid IP Address: %v", fields[0])
					continue
				}
				for i := 1; i < len(fields); i++ {
					hostsMap[fields[i]] = ip
				}
			}
		}
	}

	return hostsMap, nil
}

// ParseFile is a wrapper for the Parse() function call that takes a URL as an input and returns the hosts file
// map if successful, otherwise an error is returned.
func ParseUrl(url string) (Map, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return Parse(resp.Body)
}

// ParseFile is a wrapper for the Parse() function call that takes a filename as an input and returns the hosts file
// map if successful, otherwise an error is returned.
func ParseFile(filename string) (Map, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return Parse(file)
}
