package dialer

import (
	"net"
	"time"
	"github.com/riobard/go-shadowsocks2/socks"
)

type DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)

type ConnectionSpec interface {
	ServerWrapConn(conn net.Conn) net.Conn
	ClientWrapDial(d DialFunc) DialFunc
}

type CommonConnection interface {
	net.Conn
	Init(parent net.Conn, config interface{}) error
}

type ForwardConnection interface {
	CommonConnection
	ForwardReady() <- chan socks.Addr
}

func MakeConnection(baseConn net.Conn, connections []CommonConnection, args []interface{}) net.Conn {
	parent := baseConn

	for i, cc := range connections {
		err := cc.Init(parent, args[i])
		if err != nil {
			panic(err)
		}
		parent = cc
	}

	return parent
}