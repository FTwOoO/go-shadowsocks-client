package dialer

import "net"

type DialFunc func(network string, addr string) (net.Conn, error)
type DialMiddleware func(d DialFunc) DialFunc