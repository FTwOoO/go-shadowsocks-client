package connection

import (
	"time"
	"net"
	"errors"
	"encoding/binary"
	"sync/atomic"
	"sync"
	"log"
	"bytes"
	"io"
	"context"
	"fmt"
)

var pool = &sync.Pool{}

func init() {
	pool.New = func() interface{} {
		return make([]byte, ByteItemLen)
	}
}

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

const BufferItemsPerStream = 10
const ByteItemLen = 1024

type ByteItem []byte

var _ io.ReadWriter = &DataBuffer{}

type DataBuffer struct {
	net.Conn
	readItemsCh    chan []byte
	ReadReady      chan struct{}
	dataReadOffset uint32
	readBuffer     *bytes.Buffer
}

func NewBufferRead(DataReadOffset uint32) *DataBuffer {
	bs := new(DataBuffer)
	bs.readItemsCh = make(chan []byte, BufferItemsPerStream)
	bs.ReadReady = make(chan struct{}, 10)
	bs.readBuffer = &bytes.Buffer{}
	bs.dataReadOffset = DataReadOffset
	return bs
}

func (bs *DataBuffer) GetDataReadOffset() (n uint32) {
	return bs.dataReadOffset
}

func (bs *DataBuffer) Write(b []byte) (n int, err error) {

	for len(b) > 0 {
		item := pool.Get().([]byte)[:ByteItemLen]
		nCopy := copy(item, b)
		item = item[:nCopy]
		n += nCopy
		//TODO: set write timeout
		bs.readItemsCh <- b
	}
	return
}

func (bs *DataBuffer) Read(b []byte) (n int, err error) {
	for {
		if bs.readBuffer.Len() > 0 {
			n, err = bs.readBuffer.Read(b)
			bs.dataReadOffset += uint32(n)
			return
		}

		bs.readBuffer.Truncate(0)

		item, ok := <-bs.readItemsCh
		if !ok {
			err = errors.New("readItemsCh closed")
			return
		}

		defer pool.Put(item)

		n = copy(b, item)
		if len(item) > n {
			bs.readBuffer.Write(item[n:])
		}

		bs.dataReadOffset += uint32(n)
		return
	}
}

type MultiConnection struct {
	Connections     map[int]connectionChannel
	connectionsLock sync.Mutex

	readItemsCh chan []byte

	ConnId         int
	DataReadOffset uint32
}

type connectionChannel struct {
	Id          int
	context.Context
	CancelFunc  context.CancelFunc
	Conn        net.Conn
	ReadBuffer  *DataBuffer
	WriteBuffer *DataBuffer
}

func NewMultiConnectionById(connId int) (cc *MultiConnection) {
	cc = &MultiConnection{}
	cc.ConnId = connId
	cc.readItemsCh = make(chan []byte, BufferItemsPerStream)
	cc.Connections = make(map[int]connectionChannel)
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

	readBuffer := NewBufferRead(0)
	writeBuffer := NewBufferRead(0)

	connChannel := connectionChannel{
		Id:          time.Now().Nanosecond(),
		Conn:        innerConn,
		ReadBuffer:  readBuffer,
		WriteBuffer: writeBuffer,
	}

	connChannel.Context, connChannel.CancelFunc = context.WithCancel(context.Background())

	cc.Connections[connChannel.Id] = connChannel

	go cc.readLoop(connChannel)
	go cc.writeLoop(connChannel)

}

func (bs *MultiConnection) readLoop(connChannel connectionChannel) {

	go func() {
		_, err := io.Copy(connChannel.ReadBuffer, connChannel.Conn)
		if err != nil {
			connChannel.CancelFunc()
		}
	}()

	r := connChannel.ReadBuffer

	for {

		select {
		case <-connChannel.Done():
			bs.connectionsLock.Lock()
			delete(bs.Connections, connChannel.Id)
			bs.connectionsLock.Unlock()
			return
		case <-r.ReadReady:
			item := pool.Get().([]byte)[:ByteItemLen]
			readOffset := r.GetDataReadOffset()

			n, err := r.Read(item)
			if err != nil {
				log.Println(err)
				return
			}

			if bs.DataReadOffset >= readOffset && bs.DataReadOffset < (readOffset+uint32(n)) {
				item = item[bs.DataReadOffset-readOffset : n]
				atomic.AddUint32(&bs.DataReadOffset, uint32(len(item)))
				bs.readItemsCh <- item
			}
		}
	}
}

func (cc *MultiConnection) Read(b []byte) (n int, err error) {

	for n < len(b) {
		select {
		case item := <-cc.readItemsCh:
			n += copy(b[n:], item)
		default:
			return
		}
	}

	return
}

func (bs *MultiConnection) writeLoop(connChannel connectionChannel) {

	go func() {
		io.Copy(connChannel.Conn, connChannel.WriteBuffer)
		connChannel.CancelFunc()
	}()

	select {
	case <-connChannel.Done():
		bs.connectionsLock.Lock()
		delete(bs.Connections, connChannel.Id)
		bs.connectionsLock.Unlock()
		return
	}

}

func (cc *MultiConnection) Write(b []byte) (n int, err error) {
	var data []byte = b
	for len(data) > 0 {
		fmt.Print(pool.Get())
		item := pool.Get().([]byte)[:ByteItemLen]
		nCopy := copy(item, data)
		n += nCopy
		item = item[:nCopy]
		data = data[nCopy:]

		//TODO: deal with write timeout
		for _, xx := range cc.Connections {
			_, err = xx.WriteBuffer.Write(item)
			if err != nil {
				xx.CancelFunc()

				if len(cc.Connections) > 0 {
					err = nil
					continue
				} else {
					return
				}
			}
		}
	}

	return
}

func (cc *MultiConnection) Close() error {
	for _, cc := range cc.Connections {
		cc.CancelFunc()
	}

	close(cc.readItemsCh)
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
