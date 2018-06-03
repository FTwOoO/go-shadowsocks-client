package connection

import (
	"time"
	"net"
	"github.com/FTwOoO/kcp-go"
	"sync"
	"log"
	"context"
	"github.com/getlantern/errors"
)

func DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {

	if network == "kcp" {
		return kcp.Dial(address)
	} else {
		return net.DialTimeout(network, address, timeout)
	}
}

type MultiConnectionManager struct {
	conns map[int]*MultiConnection
	connsLock sync.RWMutex

	acceptConns chan *MultiConnection
	address net.Addr
}

func  NewMultiConnectionManager() *MultiConnectionManager {
	mc := &MultiConnectionManager{}
	mc.conns = make(map[int]*MultiConnection)
	mc.acceptConns = make(chan *MultiConnection)
	return mc
}

func (mc *MultiConnectionManager) DialTimeout(network, address string, timeout time.Duration) (cc net.Conn, err error) {

	ch := make(chan net.Conn)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn1, err := DialTimeout("tcp", address, timeout)
		if err != nil {
			return
		}

		ch <- conn1
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn2, err := DialTimeout("kcp", address, timeout)
		if err != nil {
			return
		}
		ch <- conn2
	}()

 	id := time.Now().Nanosecond()
	conn  := NewMultiConnectionById(id)
	if err != nil {
		return
	}

	mc.connsLock.Lock()
	mc.conns[id] = conn
	mc.connsLock.Unlock()

	go func() {
		wg.Wait()
		close(ch)
	}()

	cc1 := <- ch
	conn.Add(cc1)

	go func() {
		for cc1 := range ch {
			conn.Add(cc1)
		}
	}()

	cc = conn
	return
}

func (mc *MultiConnectionManager) Listen(network, address string) (l net.Listener, err error) {
	mc.address, _ = net.ResolveTCPAddr(network, address)

	l1, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}

	l2, err := net.Listen("kcp", address)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}

	ctx := context.Background()

	for _, l := range []net.Listener{l1, l2} {
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
					if c1, ok := c.(*net.TCPConn); ok {
						c1.SetKeepAlive(true)
					}

					c1 := NewInnerConnection(c, 0)
					err = c1.ReadHeader()
					if err != nil {
						log.Print(err)
						continue
					}

					mc.connsLock.RLock()
					multiConn, ok := mc.conns[c1.Id()]
					mc.connsLock.RUnlock()

					if ok {
						multiConn.Add(c1)
					} else {
						multiConn = NewMultiConnectionById(c1.Id())
						multiConn.Add(c1)

						mc.connsLock.Lock()
						mc.conns[c1.connId] = multiConn
						mc.connsLock.Unlock()

						mc.acceptConns <- multiConn

					}
				}
			}
		}()
	}

	return mc, nil

}

// Accept waits for and returns the next connection to the listener.
func (mc *MultiConnectionManager) Accept() (net.Conn, error) {
	cc, ok := <- mc.acceptConns
	if !ok {
		return nil, errors.New("closed")
	}

	return cc, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (mc *MultiConnectionManager) Close() error {
	return nil
}

// Addr returns the listener's network address.
func (mc *MultiConnectionManager) Addr() net.Addr {
	return mc.address
}
