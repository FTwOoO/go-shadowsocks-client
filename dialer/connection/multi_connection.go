package connection

import (
	"time"
	"net"
	"sync"
	"log"
	"io"
)

var pool = &sync.Pool{}

func init() {
	pool.New = func() interface{} {
		return make([]byte, ByteItemLen)
	}
}

const ByteItemLen = 1024

type MultiConnection struct {
	Connections     []*connectionChannel
	connectionsLock sync.Mutex

	ConnId         int
	ReadBuffer     *DataBuffer
	ReadBufferLock sync.Mutex
}

type connectionChannel struct {
	Id          int
	Conn        net.Conn
	ReadBuffer  *DataBuffer
	WriteBuffer *DataBuffer
}

func (cc *connectionChannel) Close() {
	cc.Conn.Close()
	cc.ReadBuffer.Close()
	cc.WriteBuffer.Close()
}

func NewMultiConnectionById(connId int) (cc *MultiConnection) {
	cc = &MultiConnection{}
	cc.ConnId = connId

	cc.ReadBuffer = NewBufferRead(ByteItemLen, 0)
	return
}

func (cc *MultiConnection) Add(conn net.Conn) {
	cc.connectionsLock.Lock()
	defer cc.connectionsLock.Unlock()

	var innerConn *InnerConnection
	innerConn, ok := conn.(*InnerConnection)
	if !ok {
		innerConn = NewInnerConnection(
			conn,
			cc.ConnId,
		)
	} else {
		if innerConn.Id() != cc.ConnId {
			log.Fatal("Id(%d) != ConnId(%d)", innerConn.Id(), cc.ConnId)
		}
	}

	connChannel := &connectionChannel{
		Id:          time.Now().Nanosecond(),
		Conn:        innerConn,
		ReadBuffer:  NewBufferRead(ByteItemLen, 0),
		WriteBuffer: NewBufferRead(ByteItemLen, 0),
	}

	cc.Connections = append(cc.Connections, connChannel)
	go cc.readLoop(connChannel.ReadBuffer, connChannel.Conn)
	go func() {
		go io.Copy(connChannel.Conn, connChannel.WriteBuffer)
		io.Copy(connChannel.ReadBuffer, connChannel.Conn)
		connChannel.Close()

	}()
}

func (bs *MultiConnection) readLoop(ReadBuffer *DataBuffer, conn net.Conn) {

	r := ReadBuffer

	for {
		item := pool.Get().([]byte)[:ByteItemLen]
		readOffset := r.GetReadOffset()

		n, err := r.Read(item)
		if err != nil {
			log.Println(err)
			return
		}

		//NEED lock
		bs.ReadBufferLock.Lock()
		writeOffset := bs.ReadBuffer.GetWriteOffset()

		if writeOffset >= readOffset && writeOffset < (readOffset+n) {
			item = item[writeOffset-readOffset : n]
			_, err := bs.ReadBuffer.Write(item)
			if err != nil {
				bs.Close()
				bs.ReadBufferLock.Unlock()
				return
			}
		}
		bs.ReadBufferLock.Unlock()
	}
}

func (cc *MultiConnection) Read(b []byte) (n int, err error) {
	return cc.ReadBuffer.Read(b)
}

func (cc *MultiConnection) Write(b []byte) (n int, err error) {

	//TODO: deal with write timeout
	for _, xx := range cc.Connections {
		_, err = xx.WriteBuffer.Write(b)
		if err != nil {

			if len(cc.Connections) > 0 {
				err = nil
				continue
			} else {
				return
			}
		}

	}

	return
}

func (cc *MultiConnection) Close() error {
	cc.ReadBuffer.Close()

	for _, x := range cc.Connections {
		x.Close()
	}
	return nil
}

func (cc *MultiConnection) LocalAddr() net.Addr {
	return cc.Connections[0].Conn.LocalAddr()
}

func (cc *MultiConnection) RemoteAddr() net.Addr {
	return cc.Connections[0].Conn.RemoteAddr()
}

func (cc *MultiConnection) SetDeadline(t time.Time) error {
	return nil
}

func (cc *MultiConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (cc *MultiConnection) SetWriteDeadline(t time.Time) error {
	return nil
}
