package serv

import (
	"net"
	"log"
	"context"
	"github.com/FTwOoO/go-ss/dialer"
	"time"
	"github.com/FTwOoO/kcp-go"
)

func KcpRemote(addr string, trans func(conn net.Conn) net.Conn, ctx context.Context) (err error) {
	l, err := kcp.Listen(addr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", addr, err)
		return
	}

	log.Printf("listening KCP on %s", addr)
	go listenForConn(l, trans, ctx)

	return
}

func TcpRemote(addr string, trans func(conn net.Conn) net.Conn, ctx context.Context) (err error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", addr, err)
		return
	}

	log.Printf("listening TCP on %s", addr)
	go listenForConn(l, trans, ctx)

	return
}

func listenForConn(l net.Listener, trans func(conn net.Conn) net.Conn, ctx context.Context) {
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

			c = trans(c)

			go forwardConnection(c)


		}
	}
}

func forwardConnection(c net.Conn) {

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
