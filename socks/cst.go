package socks

import (
	"errors"
	"net"
	"io"
)

const (
	SocksVer4       = 4
	SocksVer5       = 5
	SocksCmdConnect = 1

	AddrTypeIPv4 = 1 // type is ipv4 address
	AddrTypeDomain   = 3 // type is domain address
	AddrTypeIPv6 = 4 // type is ipv6 address
)

var (
	errAddrType      = errors.New("socks addr type not supported")
	errVer           = errors.New("socks version not supported")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks command not supported")
	errReject = errors.New("socks reject this request")
	errSupported     = errors.New("proxy type not supported")
)

func HandleSocksConnection(conn net.Conn, cb func(host string, addrType int, conn net.Conn) error ) (err error) {
	isClose := false
	defer func() {
		if !isClose {
			conn.Close()
		}
	}()
	var (
		host     string
		hostType int
	)

	buf := make([]byte, 1)
	io.ReadFull(conn, buf)

	first := buf[0]
	switch first {
	case SocksVer5:
		err = handshake(conn, first)
		if err != nil {
			return
		}
		host, hostType, err = socks5Connect(conn)
		return cb(host, hostType, conn)
	case SocksVer4:
		host, hostType, err = socks4Connect(conn, first)
		return cb(host, hostType, conn)

	default:
		return errVer
	}

}
