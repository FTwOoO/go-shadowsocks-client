package connection

import (
	"net"
	"encoding/binary"
)

type InnerConnection struct {
	net.Conn
	connId  int
	isRead  bool
	isWrite bool
}

func NewInnerConnection(conn net.Conn, id int) *InnerConnection {
	return &InnerConnection{Conn: conn, connId: id}
}

func (cc *InnerConnection) Id() (int) {
	return cc.connId
}

func (cc *InnerConnection) ReadHeader() (err error) {
	if !cc.isRead {
		err = binary.Read(cc.Conn, binary.BigEndian, cc.connId)
		if err != nil {
			return
		}

		cc.isRead = true
	}

	return nil
}

func (cc *InnerConnection) Read(b []byte) (n int, err error) {
	if !cc.isRead {
		err = cc.ReadHeader()
		if err != nil {
			return
		}
	}

	return cc.Conn.Read(b)
}

func (cc *InnerConnection) Write(b []byte) (n int, err error) {
	if !cc.isWrite {
		err = binary.Write(cc.Conn, binary.BigEndian, cc.connId)
		if err != nil {
			return
		}

		cc.isWrite = true
	}

	return cc.Conn.Write(b)
}
