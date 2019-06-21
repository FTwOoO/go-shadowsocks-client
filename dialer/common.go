package dialer

import (
	"context"
	"github.com/FTwOoO/go-ss/socks"
	"net"
	"time"
)

type DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)

type ProxyProtocol interface {
	ServerListen(
		addr string,
		listenFunc func(net, laddr string) (net.Listener, error),
		handler func(ForwardConnection),
		ctx context.Context,
	) (err error)

	ClientWrapDial(d DialFunc) DialFunc
}

type CommonConnection interface {
	net.Conn
	Init(parent net.Conn, config interface{}) error
}

type ForwardConnection interface {
	CommonConnection
	ForwardReady() <-chan socks.Addr
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
