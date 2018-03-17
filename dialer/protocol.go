package dialer

import (
	"net"
	"log"
	"github.com/riobard/go-shadowsocks2/socks"
	"time"
)

type PrococolConfig struct {
	Cipher     string
	Password   string
	ServerAddr string
}

func (s *PrococolConfig) GenServerConn(conn net.Conn) net.Conn {
	ciphConn := &CipherConn{}
	ciphConn.Init(conn, &CipherConnParams{s.Cipher, s.Password})

	shadowsocksConn := &ShadowsocksRawConn{}
	shadowsocksConn.Init(ciphConn, &ShadowsocksRawConnParams{IsServer:true})
	return shadowsocksConn
}


func (s *PrococolConfig) GenClientDialer(parentDial DialFunc) DialFunc {
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
		ciphConn.Init(rc, &CipherConnParams{s.Cipher, s.Password})

		shadowsocksConn := &ShadowsocksRawConn{}
		shadowsocksConn.Init(ciphConn, &ShadowsocksRawConnParams{tgt, false})
		conn = shadowsocksConn
		return
	}
}
