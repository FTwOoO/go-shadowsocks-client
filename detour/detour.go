/*
Package detour provides a net.Conn interface to dial another dialer if a site fails to connect directly.
It maintains three states of a connection: initial, direct and detoured
along with a temporary whitelist across connections.
It also add a blocked site to permanent whitelist.

The action taken and state transistion in each phase is as follows:
+-------------------------+-----------+-------------+-------------+-------------+-------------+
|                         | no error  |   timeout*  | conn reset/ | content     | other error |
|                         |           |             | dns hijack  | hijack      |             |
+-------------------------+-----------+-------------+-------------+-------------+-------------+
| dial (intial)           | noop      | detour      | detour      | n/a         | noop        |
| first read (intial)     | direct    | detour(buf) | detour(buf) | detour(buf) | noop        |
|                         |           | add to tl   | add to tl   | add to tl   |             |
| follow-up read (direct) | direct    | add to tl   | add to tl   | add to tl   | noop        |
| follow-up read (detour) | noop      | rm from tl  | rm from tl  | rm from tl  | rm from tl  |
| close (direct)          | noop      | n/a         | n/a         | n/a         | n/a         |
| close (detour)          | add to wl | n/a         | n/a         | n/a         | n/a         |
+-------------------------+-----------+-------------+-------------+-------------+-------------+
| next dial/read(in tl)***| noop      | rm from tl  | rm from tl  | rm from tl  | rm from tl  |
| next close(in tl)       | add to wl | n/a         | n/a         | n/a         | n/a         |
+-------------------------+-----------+-------------+-------------+-------------+-------------+
(buf) = resend buffer
tl = temporary whitelist
wl = permanent whitelist

* Operation will time out in TimeoutToDetour in initial state,
but at system default or caller supplied deadline for other states;
** DNS hijack is only checked at dial time.
*** Connection is always detoured if the site is in tl or wl.
*/
package detour

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getlantern/golog"
	"github.com/FTwOoO/go-shadowsocks-client/dialer"
)

// if dial or read exceeded this timeout, we consider switch to detour
// The value depends on OS and browser and defaults to 3s
// For Windows XP, find TcpMaxConnectRetransmissions in
// http://support2.microsoft.com/default.aspx?scid=kb;en-us;314053
var TimeoutToDetour = 3 * time.Second

// if DirectAddrCh is set, when a direct connection is closed without any error,
// the connection's remote address (in host:port format) will be send to it
var DirectAddrCh chan string = make(chan string)

var (
	log = golog.LoggerFor("detour")

	// instance of Detector
	blockDetector atomic.Value
)

func init() {
	blockDetector.Store(detectorByCountry(""))
}

type Conn struct {
	// keep track of the total bytes read in this connection
	// Keep it at the top to make sure 64-bit alignment, see
	// https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	readBytes int64

	muConn sync.RWMutex
	// the actual connection, will change so protect it
	// can't user atomic.Value as the concrete type may vary
	conn net.Conn

	// don't access directly, use inState() and setState() instead
	state uint32

	// the function to dial detour if the site fails to connect directly
	dialDetour dialer.DialFunc

	network, addr string
}

const (
	stateInitial uint32 = iota
	stateDirect
	stateDetour
)

var statesDesc = []string{
	"initially",
	"directly",
	"detoured",
}

func GenDialer(proxyDial dialer.DialFunc, directDial dialer.DialFunc) dialer.DialFunc {
	return func(network, addr string, timeout time.Duration) (conn net.Conn, err error) {
		dc := &Conn{dialDetour: proxyDial, network: network, addr: addr}
		if !whitelisted(addr) {
			log.Tracef("Attempting direct connection for %v", addr)
			detector := blockDetector.Load().(*Detector)
			dc.setState(stateDirect)
			// always try direct connection first
			dc.conn, err = directDial(network, addr, TimeoutToDetour)
			if err == nil {
				return dc, nil
			} else if detector.TamperingSuspected(err) {
				log.Debugf("Dial %s to %s failed[%s], try detour", dc.stateDesc(), addr, err)
			} else {
				log.Debugf("Dial %s to %s failed: %s", dc.stateDesc(), addr, err)
				return dc, err
			}
		}
		log.Tracef("Detouring %v", addr)
		// if whitelisted or dial directly failed, try detour
		dc.setState(stateDetour)
		dc.conn, err = dc.dialDetour(network, addr, TimeoutToDetour)
		if err != nil {
			log.Errorf("Dial %s failed: %s", dc.stateDesc(), err)
			return nil, err
		}
		log.Tracef("Dial %s to %s succeeded", dc.stateDesc(), addr)
		if !whitelisted(addr) {
			log.Tracef("Add %s to whitelist", addr)
			AddToWhiteList(dc.addr, false)
		}
		return dc, err
	}
}

func (dc *Conn) Read(b []byte) (n int, err error) {
	detector := blockDetector.Load().(*Detector)

	if n, err = dc.countedRead(b); err != nil {
		if err == io.EOF {
			log.Tracef("Read %d bytes from %s %s, EOF", n, dc.addr, dc.stateDesc())
			return
		}
		log.Tracef("Read from %s %s failed: %s", dc.addr, dc.stateDesc(), err)
		switch {
		case dc.inState(stateDirect) && detector.TamperingSuspected(err):
			// to prevent a slow or unstable site from been treated as blocked,
			// we only check first 4K bytes, which roughly equals to the payload of 3 full packets on Ethernet
			if atomic.LoadInt64(&dc.readBytes) <= 4096 {
				log.Tracef("Seems %s still blocked, add to whitelist so will try detour next time", dc.addr)
				AddToWhiteList(dc.addr, false)
			}
		case dc.inState(stateDetour) && wlTemporarily(dc.addr):
			log.Tracef("Detoured route is not reliable for %s, not whitelist it", dc.addr)
			RemoveFromWl(dc.addr)
		}
		return
	}

	log.Tracef("Read %d bytes from %s %s", n, dc.addr, dc.stateDesc())
	return
}

// Write() implements the function from net.Conn
func (dc *Conn) Write(b []byte) (n int, err error) {
	if n, err = dc.getConn().Write(b); err != nil {
		log.Debugf("Error while write %d bytes to %s %s: %s", len(b), dc.addr, dc.stateDesc(), err)
		return
	}
	log.Tracef("Wrote %d bytes to %s %s", len(b), dc.addr, dc.stateDesc())
	return
}

// Close() implements the function from net.Conn
func (dc *Conn) Close() error {
	log.Tracef("Closing %s connection to %s", dc.stateDesc(), dc.addr)
	if atomic.LoadInt64(&dc.readBytes) > 0 {
		if dc.inState(stateDetour) && wlTemporarily(dc.addr) {
			log.Tracef("no error found till closing, add %s to permanent whitelist", dc.addr)
			AddToWhiteList(dc.addr, true)
		} else if dc.inState(stateDirect) && !wlTemporarily(dc.addr) {
			log.Tracef("no error found till closing, notify caller that %s can be dialed directly", dc.addr)
			// just fire it, but not blocking if the chan is nil or no reader
			select {
			case DirectAddrCh <- dc.addr:
			default:
			}
		}
	}
	conn := dc.getConn()
	if conn == nil {
		return nil
	}
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
		log.Debugf("Unable to set read deadline: %v", err)
	}
	if err := dc.SetWriteDeadline(t); err != nil {
		log.Debugf("Unable to set write deadline: %v", err)
	}
	return nil
}

func (dc *Conn) SetReadDeadline(t time.Time) error {
	if err := dc.getConn().SetReadDeadline(t); err != nil {
		log.Debugf("Unable to set read deadline: %v", err)
	}
	return nil
}

func (dc *Conn) SetWriteDeadline(t time.Time) error {
	if err := dc.getConn().SetWriteDeadline(t); err != nil {
		log.Debugf("Unable to set write deadline", err)
	}
	return nil
}

func (dc *Conn) countedRead(b []byte) (n int, err error) {
	n, err = dc.getConn().Read(b)
	atomic.AddInt64(&dc.readBytes, int64(n))
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

	log.Tracef("Replaced connection to %s from direct to detour and closing old one", dc.addr)
	if err := oldConn.Close(); err != nil {
		log.Debugf("Unable to close old connection: %v", err)
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
