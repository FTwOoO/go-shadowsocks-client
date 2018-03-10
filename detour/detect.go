package detour

import (
	"net"
)

// Detector is just a set of rules to check if a site is potentially blocked or not
type Detector struct {
	DNSPoisoned        func(net.Conn) bool
	TamperingSuspected func(error) bool
}

var detectors = make(map[string]*Detector)

var defaultDetector = Detector{
	DNSPoisoned: func(net.Conn) bool { return false },
	TamperingSuspected: func(err error) bool {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return true
		}
		if _, ok := err.(*net.OpError); ok {
			// Let's be more aggressive on considering which errors are the
			// symptom of being blocked, because we can't reliably enumerate all
			// relevant errors. It's also a big plus if Lantern can help user to
			// bypass those various network errors, e.g., unresolvable host, route
			// errors, accessing IPv6 host from IPv4 network, etc.
			return true
		}
		return false
	},
}

func detectorByCountry(country string) *Detector {
	d := detectors[country]
	if d == nil {
		return &defaultDetector
	}
	return &Detector{
		d.DNSPoisoned,
		func(err error) bool {
			return defaultDetector.TamperingSuspected(err) || d.TamperingSuspected(err)
		},
	}
}
