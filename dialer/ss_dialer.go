package dialer

import (
	"net"
	"log"
	"github.com/riobard/go-shadowsocks2/core"
	"encoding/base64"
	"github.com/riobard/go-shadowsocks2/socks"
)

type Shadowsocks struct {
	Cipher     string
	Password   string
	Key        string
	ServerAddr string

	ciph func(net.Conn) net.Conn
}

func (s *Shadowsocks) getCipherStream() (ciph func(net.Conn) net.Conn, err error) {
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

func (s *Shadowsocks) GenDialer(parentDial DialFunc) DialFunc {
	return func(network string, addr string) (conn net.Conn, err error) {
		var ciph func(net.Conn) net.Conn
		ciph, err = s.getCipherStream()
		if err != nil {
			return
		}

		rc, err := parentDial("tcp", s.ServerAddr)
		if err != nil {
			log.Printf("failed to connect to server %v: %v", s.ServerAddr, err)
			return
		}
		if rc2, ok := rc.(*net.TCPConn); ok {
			rc2.SetKeepAlive(true)
		}

		rc = ciph(rc)

		tgt := socks.ParseAddr(addr)

		if tgt == nil {
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
