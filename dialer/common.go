package dialer

import (
	"net"
	"time"
)

type DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)
type DialMiddleware func(d DialFunc) DialFunc