package serv

import (
	"io"
	"net"
	"time"

	"github.com/riobard/go-shadowsocks2/socks"
	"log"
	"github.com/FTwOoO/go-shadowsocks-client/dialer"
)

func SocksLocal(socksAddr string, dial dialer.DialFunc) {
	l, err := net.Listen("tcp", socksAddr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", socksAddr, err)
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

			tgt, err := socks.Handshake(c)
			if err != nil {
				log.Printf("failed to get target address: %v", err)
				return
			}

			rc, err := dial("tcp", tgt.String())
			if err != nil {
				log.Printf("failed to connect to server %v: %v", tgt.String(), err)
				return
			}
			defer rc.Close()

			log.Printf("🏄‍ %s <-> %s", c.RemoteAddr(), tgt.String())

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
