package hosts

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"strings"
)

type Map map[string]string

func clearComments(s string) string {
	parts := strings.Split(s, "#")
	return parts[0]
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
				for i := 1; i < len(fields); i++ {
					hostsMap[fields[i]] = fields[0]
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
