package serv

import (
	"net"
	"github.com/riobard/go-shadowsocks2/socks"
	"log"
	"context"
	"github.com/FTwOoO/go-ss/dialer"
	"time"
)

func TcpRemote(addr string, trans func(conn net.Conn) net.Conn, ctx context.Context) (err error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", addr, err)
		return
	}

	log.Printf("listening TCP on %s", addr)

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
				c = trans(c)

				select {
				case <-time.After(5*time.Second):
					log.Printf("timeout for connection to receive target: %s", c.RemoteAddr())
					c.Close()
				case addr := <-c.(dialer.ForwardConnection).ForwardReady():
					go forwardConnection(c, addr)
				}
			}
		}
	}()

	return
}

func forwardConnection(c net.Conn, tgt socks.Addr) {
	defer c.Close()
	c.(*net.TCPConn).SetKeepAlive(true)

	rc, err := net.Dial("tcp", tgt.String())
	if err != nil {
		log.Printf("failed to connect to target: %v", err)
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
}

