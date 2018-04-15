package dialer

import (
	"net"
	"log"
	"github.com/riobard/go-shadowsocks2/socks"
	"time"
)

type SSPrococolConfig struct {
	Cipher     string
	Password   string
	ServerAddr string  //client only
}

func (s *SSPrococolConfig) GenServerConn(conn net.Conn) net.Conn {
	ciphConn := &CipherConn{}
	err := ciphConn.Init(conn, CipherConnParams{s.Cipher, s.Password})
	if err != nil {
		panic(err)
	}

	shadowsocksConn := &ShadowsocksRawConn{}
	err = shadowsocksConn.Init(ciphConn, ShadowsocksRawConnParams{IsServer:true})
	if err != nil {
		panic(err)
	}
	return shadowsocksConn
}


func (s *SSPrococolConfig) GenClientDialer(parentDial DialFunc) DialFunc {
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

		ciphConn := &CipherConn{}
		err = ciphConn.Init(rc, CipherConnParams{s.Cipher, s.Password})
		if err != nil {
			return
		}

		shadowsocksConn := &ShadowsocksRawConn{}
		params := ShadowsocksRawConnParams{tgt, false}
		err = shadowsocksConn.Init(ciphConn, params)
		if err != nil {
			return
		}
		conn = shadowsocksConn
		return
	}
}
