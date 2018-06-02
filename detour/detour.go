package detour

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"log"
	"github.com/FTwOoO/go-ss/dialer"
)

var (
	detector *Detector   = &Detector{SiteStat: siteStat}
	rules    DetourRules = &CNRules{}
)

type Conn struct {
	// keep track of the total bytes read in this connection
	// Keep it at the top to make sure 64-bit alignment, see
	// https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	readBytes int64

	lastError error

	muConn sync.RWMutex
	// the actual connection, will change so protect it
	// can't user atomic.Value as the concrete type may vary
	conn net.Conn

	// don't access directly, use inState() and setState() instead
	state uint32

	// the function to dial detour if the site fails to connect directly
	dialDetour dialer.DialFunc

	network, addr string
	host          string
}

const (
	stateInitial uint32 = iota
	stateDirect
	stateProxy
)

var statesDesc = []string{
	"initial",
	"direct",
	"proxy",
}

func GenDial(proxyDial dialer.DialFunc, directDial dialer.DialFunc) dialer.DialFunc {
	return func(network, addr string, timeout time.Duration) (conn net.Conn, err error) {
		dc := &Conn{dialDetour: proxyDial, network: network, addr: addr}
		dc.host, _, _ = net.SplitHostPort(dc.addr)
		rule := dc.GetRule()

		try_times := 0

		if rule == AlwaysDirect || (rule == AutoTry && detector.TryDirect(dc.host)) {
			goto DirectDial
		} else {
			goto ProxyDial
		}


	DirectDial:
		try_times += 1
		log.Printf("try direct %v", addr)
		dc.setState(stateDirect)
		dc.conn, err = directDial(network, addr, defaultDialTimeout)
		if err == nil {
			return dc, nil
		}

		log.Printf("direct to %s failed[%s]", addr, err)
		if try_times >= 2 || rule == AlwaysDirect {
			return
		} else {
			goto ProxyDial
		}

	ProxyDial:
		try_times += 1
		log.Printf("try proxy %v", addr)
		dc.setState(stateProxy)
		dc.conn, err = dc.dialDetour(network, addr, defaultDialTimeout)
		if err == nil {
			log.Printf("proxy to %s succeeded", addr)
			return dc, nil
		}

		log.Printf("proxy to %s failed[%s]", addr, err)
		if try_times >= 2 || rule == AlwaysProxy {
			return
		} else {
			goto DirectDial
		}
	}
}

func (dc *Conn) GetRule() (r DetourStrategy) {
	r = rules.GetRule(dc.host)
	return
}

func (dc *Conn) Read(b []byte) (n int, err error) {

	err = dc.SetReadDeadline(time.Now().Add(defaultReadTimeout))
	if err != nil {
		log.Printf("set readdeadline error:%s", err)
		return
	}

	if n, err = dc.countedRead(b); err != nil {
		if err == io.EOF {
			log.Printf("Read %d bytes from %s %s, EOF", n, dc.addr, dc.stateDesc())
			return
		}
		//log.Printf("Read from %s %s failed: %s", dc.addr, dc.stateDesc(), err)
		bytes := atomic.LoadInt64(&dc.readBytes)

		if !detector.IsTimeout(err) && bytes > 0 && bytes <= 4096 {
			// to prevent a slow or unstable site from been treated as blocked,
			// we only check first 4K bytes, which roughly equals to the payload of 3 full packets on Ethernet
			dc.lastError = err
			log.Printf("Read from %s %s bytes[%d] timeout, treat as error", dc.addr, dc.stateDesc(), bytes)
		}

		return
	}

	//log.Printf("Read %d bytes from %s %s", n, dc.addr, dc.stateDesc())
	return
}

func (dc *Conn) Write(b []byte) (n int, err error) {
	if n, err = dc.getConn().Write(b); err != nil {
		log.Printf("Error while write %d bytes to %s %s: %s", len(b), dc.addr, dc.stateDesc(), err)
		return
	}
	//log.Printf("Wrote %d bytes to %s %s", len(b), dc.addr, dc.stateDesc())
	return
}

func (dc *Conn) Close() error {
	log.Printf("close %s connection to %s", dc.stateDesc(), dc.addr)

	if dc.GetRule() == AutoTry {
		if dc.lastError == nil {
			switch {
			case dc.inState(stateDirect):
				log.Printf("save direct for %s", dc.addr)
				detector.DirectVisitSuccess(dc.host)

			case dc.inState(stateProxy):
				log.Printf("save proxy for %s", dc.addr)
				detector.BlockedVisitSuccess(dc.host)
			}
		}
	}

	conn := dc.getConn()
	return conn.Close()
}

func (dc *Conn) LocalAddr() net.Addr {
	return dc.getConn().LocalAddr()
}

func (dc *Conn) RemoteAddr() net.Addr {
	return dc.getConn().RemoteAddr()
}

func (dc *Conn) SetDeadline(t time.Time) error {
	if err := dc.SetReadDeadline(t); err != nil {
		log.Printf("Unable to set read deadline: %v", err)
	}
	if err := dc.SetWriteDeadline(t); err != nil {
		log.Printf("Unable to set write deadline: %v", err)
	}
	return nil
}

func (dc *Conn) SetReadDeadline(t time.Time) error {
	if err := dc.getConn().SetReadDeadline(t); err != nil {
		log.Printf("Unable to set read deadline: %v", err)
	}
	return nil
}

func (dc *Conn) SetWriteDeadline(t time.Time) error {
	if err := dc.getConn().SetWriteDeadline(t); err != nil {
		log.Printf("Unable to set write deadline", err)
	}
	return nil
}

func (dc *Conn) countedRead(b []byte) (n int, err error) {
	n, err = dc.getConn().Read(b)
	if err == nil {
		atomic.AddInt64(&dc.readBytes, int64(n))
	}
	return
}

func (dc *Conn) getConn() (c net.Conn) {
	dc.muConn.RLock()
	defer dc.muConn.RUnlock()
	return dc.conn
}

func (dc *Conn) setConn(c net.Conn) {
	dc.muConn.Lock()
	oldConn := dc.conn
	dc.conn = c
	dc.muConn.Unlock()

	log.Printf("Replaced connection to %s from direct to detour and closing old one", dc.addr)
	if err := oldConn.Close(); err != nil {
		log.Printf("Unable to close old connection: %v", err)
	}
}

func (dc *Conn) stateDesc() string {
	return statesDesc[atomic.LoadUint32(&dc.state)]
}

func (dc *Conn) inState(s uint32) bool {
	return atomic.LoadUint32(&dc.state) == s
}

func (dc *Conn) setState(s uint32) {
	atomic.StoreUint32(&dc.state, s)
}
