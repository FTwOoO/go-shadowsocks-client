package serv

import (
	"io"
	"net"
	"time"

	"github.com/riobard/go-shadowsocks2/socks"
	"log"
)

// Create a SOCKS server listening on addr and proxy to server.
func SocksLocal(addr, server string, shadow func(net.Conn) net.Conn) {
	log.Printf("SOCKS proxy %s <-> %s", addr, server)
	TcpLocal(addr, server, shadow, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
}

// Listen on addr and proxy to server to reach target from getAddr.
func TcpLocal(addr, server string, shadow func(net.Conn) net.Conn, getAddr func(net.Conn) (socks.Addr, error)) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", addr, err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("failed to accept: %s", err)
			continue
		}

		go func() {
			defer c.Close()
			c.(*net.TCPConn).SetKeepAlive(true)

			tgt, err := getAddr(c)
			if err != nil {
				log.Printf("failed to get target address: %v", err)
				return
			}

			rc, err := net.Dial("tcp", server)
			if err != nil {
				log.Printf("failed to connect to server %v: %v", server, err)
				return
			}
			defer rc.Close()
			rc.(*net.TCPConn).SetKeepAlive(true)
			rc = shadow(rc)

			if _, err = rc.Write(tgt); err != nil {
				log.Printf("failed to send target address: %v", err)
				return
			}

			log.Printf("proxy %s <-> %s <-> %s", c.RemoteAddr(), server, tgt)
			_, _, err = relay(rc, c)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					return // ignore i/o timeout
				}
				log.Printf("relay error: %v", err)
			}
		}()
	}
}

// relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
func relay(left, right net.Conn) (int64, int64, error) {
	type res struct {
		N   int64
		Err error
	}
	ch := make(chan res)

	go func() {
		n, err := io.Copy(right, left)
		right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
		left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
		ch <- res{n, err}
	}()

	n, err := io.Copy(left, right)
	right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
	left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	rs := <-ch

	if err == nil {
		err = rs.Err
	}
	return n, rs.N, err
}
