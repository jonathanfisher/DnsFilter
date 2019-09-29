// statistics is a package intended to centralize the logic for tracking statistics on DNS requests and responses.
package statistics

import (
	"log"
	"net"
	"time"
)

type Event struct {
	Client        net.IP
	NameRequested string
	IPResponse    net.IP
	Timestamp     time.Time
}

type Statistics struct {
	Blocked map[string]int
	Allowed map[string]int
	Channel chan Event
}

// New will create a new statistics structure that is ready for use. Note that this will start a goroutine that
// listens on the newly-created channel for messages to process.
func New() *Statistics {
	var stats *Statistics

	stats = new(Statistics)
	stats.Channel = make(chan Event)

	go stats.logLoop()

	return stats
}

// logLoop is intended to be run as a goroutine, and is responsible for popping off of the internal messaging channel
// and logging accordingly.
func (s *Statistics) logLoop() {
	for {
		event := <- s.Channel
		log.Printf("Client: %v, Requested: %v, IP Response: %v, Timestamp: %v",
			event.Client, event.NameRequested, event.IPResponse, event.Timestamp)
	}
}

// LogEvent is a simple wrapper function that takes a logging event structure and adds it to the internal messaging
// channel for processing by the logLoop() goroutine above.
func (s *Statistics) LogEvent(e Event) {
	s.Channel <- e
}
