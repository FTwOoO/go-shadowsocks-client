package dialer

import (
	"net"
	"log"
	"github.com/riobard/go-shadowsocks2/core"
	"encoding/base64"
	"github.com/riobard/go-shadowsocks2/socks"
	"fmt"
)
type DialFunc func(network string, addr string) (net.Conn, error)
type DialMiddleware func(d DialFunc) DialFunc

type ShadowsocksDialer struct {
	Cipher string
	Password string
	Key string
	ServerAddr string

	ciph func (net.Conn) net.Conn
}

func (s * ShadowsocksDialer) getCipherStream() (ciph func (net.Conn) net.Conn, err error) {
	if s.ciph != nil {
		return s.ciph, nil
	}

	var key []byte
	if s.Key != "" {
		key, err = base64.URLEncoding.DecodeString(s.Key)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	var ss core.Cipher
	ss, err = core.PickCipher(s.Cipher, key, s.Password)
	if err != nil {
		log.Fatal(err)
		return
	}
	ciph = ss.StreamConn
	s.ciph = ciph
	return
}

func (s * ShadowsocksDialer) Dialer(d DialFunc) DialFunc {
	return func(network string, addr string) (conn net.Conn, err error) {
		var ciph func (net.Conn) net.Conn
		ciph, err = s.getCipherStream()
		if err != nil {
			return
		}

		rc, err := d("tcp", s.ServerAddr)
		if err != nil {
			log.Printf("failed to connect to server %v: %v", s.ServerAddr, err)
			return
		}
		defer rc.Close()
		rc.(*net.TCPConn).SetKeepAlive(true)
		rc = ciph(rc)

		tgt, err := addr2SocksAddr(addr)
		if err != nil {
			log.Printf("Invalid address: %s %v", addr, err)
			return
		}

		if _, err = rc.Write(tgt); err != nil {
			log.Printf("failed to send target address: %v", err)
			return
		}

		conn = rc
		return
	}
}


func addr2SocksAddr(addr string) (socks.Addr, error) {
	//host, port, err := net.SplitHostPort(addr)
	//if err != nil {
	//	return nil, err
	//}
	//
	//ip := net.ParseIP(host)
	//if ip != nil {
	//	return socks.Addr([]byte{})
	//}

	fmt.Println(addr)
}
