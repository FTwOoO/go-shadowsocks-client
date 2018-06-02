package protocol

import (
	"net"
	"log"
	"github.com/riobard/go-shadowsocks2/socks"
	"time"
	"github.com/FTwOoO/go-ss/dialer/connection"
	"github.com/FTwOoO/go-ss/dialer"

)

var _ dialer.ConnectionSpec = &SSProxyPrococol{}

type SSProxyPrococol struct {
	Cipher     string
	Password   string
	ServerAddr string //client only
}

func (s *SSProxyPrococol) ServerWrapConn(conn net.Conn) net.Conn {

	return dialer.MakeConnection(conn,
		[]dialer.CommonConnection{
			&connection.CipherConn{},
			&connection.ShadowsocksRawConn{},
		},
		[]interface{}{
			connection.CipherConnParams{s.Cipher, s.Password},
			connection.ShadowsocksRawConnParams{IsServer: true},
		})
}

func (s *SSProxyPrococol) ClientWrapDial(parentDial dialer.DialFunc) dialer.DialFunc {

	return func(network, addr string, timeout time.Duration) (conn net.Conn, err error) {

		rc, err := parentDial("tcp", s.ServerAddr, timeout)
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
