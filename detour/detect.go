package detour

import (
	"net"
)

// Detector is just a set of rules to check if a site is potentially blocked or not
type Detector struct {
	IsTimeout func(error) bool
}
var defaultDetector = Detector{
	IsTimeout: func (err error) bool {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return true
		}

		return false
	},
}
