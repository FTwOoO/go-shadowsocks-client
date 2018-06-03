package protocol

import (
	"net"
	"log"
	"github.com/riobard/go-shadowsocks2/socks"
	"time"
	"github.com/FTwOoO/go-ss/dialer/connection"
	"github.com/FTwOoO/go-ss/dialer"
	"context"
)

var _ dialer.ProxyProtocol = &SSProxyPrococol{}

type SSProxyPrococol struct {
	Cipher     string
	Password   string
	ServerAddr string //client only
}

func (s *SSProxyPrococol) serverWrapConn(conn net.Conn) dialer.ForwardConnection {

	return dialer.MakeConnection(conn,
		[]dialer.CommonConnection{
			&connection.CipherConn{},
			&connection.ShadowsocksRawConn{},
		},
		[]interface{}{
			connection.CipherConnParams{s.Cipher, s.Password},
			connection.ShadowsocksRawConnParams{IsServer: true},
		}).(dialer.ForwardConnection)
}

func (s *SSProxyPrococol) ServerListen(
	addr string,
	listenFunc func(net, laddr string) (net.Listener, error),
	handler func(dialer.ForwardConnection),
	ctx context.Context,
) (err error) {

	l, err := listenFunc("tcp", addr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", addr, err)
		return
	}

	log.Printf("listening on %s", addr)
	go func() {
		for {
			select {
			case <-ctx.Done():
				l.Close()
				return
			default:
				c, err := l.Accept()
				if err != nil {
					log.Printf("failed to accept: %s", err)
					continue
				}
				if c1, ok := c.(*net.TCPConn); ok {
					c1.SetKeepAlive(true)
				}

				c2 := s.serverWrapConn(c)
				if handler == nil {
					handler = forwardConnection
				}
				go handler(c2)
			}
		}
	}()

	return
}

func (s *SSProxyPrococol) ClientWrapDial(transportDial dialer.DialFunc) dialer.DialFunc {

	return func(network, addr string, timeout time.Duration) (conn net.Conn, err error) {

		rc, err := transportDial("tcp", s.ServerAddr, timeout)
		if err != nil {
			log.Printf("failed to connect to server %v: %v", s.ServerAddr, err)
			return
		}
		if rc2, ok := rc.(*net.TCPConn); ok {
			rc2.SetKeepAlive(true)
		}

		tgt := socks.ParseAddr(addr)

		if tgt == nil {
			log.Printf("Invalid address: %s %v", addr, err)
			return
		}

		conn = dialer.MakeConnection(rc,
			[]dialer.CommonConnection{
				&connection.CipherConn{},
				&connection.ShadowsocksRawConn{},
			},
			[]interface{}{
				connection.CipherConnParams{s.Cipher, s.Password},
				connection.ShadowsocksRawConnParams{Target: tgt, IsServer: false},
			})
		return
	}
}


func forwardConnection(c dialer.ForwardConnection) {

	go func() {
		select {
		case tgt := <-c.(dialer.ForwardConnection).ForwardReady():
			defer c.Close()

			rc, err := net.Dial("tcp", tgt.String())
			if err != nil {
				log.Printf("failed to connect to target: %v", err)
				return
			}

			defer rc.Close()
			log.Printf("🏄‍ %s <-tunnel-> %s <-forward-> %s", c.RemoteAddr(), c.LocalAddr(), tgt.String())
			_, _, err = relay(rc, c)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					return // ignore i/o timeout
				}
				log.Printf("relay error: %v", err)
			}

		case <-time.After(5 * time.Second):
			log.Printf("timeout for connection(%s) <-> %s", c.RemoteAddr(), c.LocalAddr())
			c.Close()
		}

	}()

	c.Read(nil)
}
