package connection

import (
	"fmt"
	"github.com/FTwOoO/go-ss/core"
	"github.com/FTwOoO/go-ss/dialer"
	"log"
	"net"
	"time"
)

type CipherConnParams struct {
	Cipher   string
	Password string
}

func (s CipherConnParams) GetCipherStream() (ciph func(net.Conn) net.Conn, err error) {
	var ss core.Cipher
	ss, err = core.PickCipher(s.Cipher, []byte{}, s.Password)
	if err != nil {
		log.Fatal(err)
		return
	}
	ciph = ss.StreamConn
	return
}

var _ dialer.CommonConnection = &CipherConn{}

type CipherConn struct {
	Conn     net.Conn
	wrapConn net.Conn
	params   CipherConnParams
}

func (cc *CipherConn) Init(parent net.Conn, args interface{}) (err error) {
	if v, ok := args.(CipherConnParams); ok {
		cc.params = v
		cc.Conn = parent
		wrapConnFunc, err := cc.params.GetCipherStream()
		if err != nil {
			return err
		}

		cc.wrapConn = wrapConnFunc(cc.Conn)
		return nil
	}

	return fmt.Errorf("args is not CipherConnParams:%s", args)
}

func (cc *CipherConn) Read(b []byte) (n int, err error) {
	return cc.wrapConn.Read(b)
}

func (cc *CipherConn) Write(b []byte) (n int, err error) {
	return cc.wrapConn.Write(b)
}

func (cc *CipherConn) Close() error {
	return cc.wrapConn.Close()
}

func (cc *CipherConn) LocalAddr() net.Addr {
	return cc.wrapConn.LocalAddr()
}

func (cc *CipherConn) RemoteAddr() net.Addr {
	return cc.wrapConn.RemoteAddr()
}

func (cc *CipherConn) SetDeadline(t time.Time) error {
	return cc.wrapConn.SetReadDeadline(t)
}

func (cc *CipherConn) SetReadDeadline(t time.Time) error {
	return cc.wrapConn.SetReadDeadline(t)
}

func (cc *CipherConn) SetWriteDeadline(t time.Time) error {
	return cc.wrapConn.SetWriteDeadline(t)
}
