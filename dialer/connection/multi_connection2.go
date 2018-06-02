package connection

import (
	"time"
	"net"
	"github.com/FTwOoO/kcp-go"
	"errors"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"sync"
	"log"
	"bytes"
	"io"
	"context"
)

var pool = &sync.Pool{}

func init() {
	pool.New = func() interface{} {
		return make([]byte, ByteItemLen)
	}
}

func DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {

	if network == "kcp" {
		return kcp.Dial(address)
	} else {
		return net.DialTimeout(network, address, timeout)
	}
}

func MutliConnDialTimeout(network, address string, timeout time.Duration) (cc net.Conn, err error) {
	if network != "tcp" {
		return nil, errors.New("Only support tcp")
	}
	conn1, err := DialTimeout("tcp", address, timeout)
	if err != nil {
		return
	}

	conn2, err := DialTimeout("kcp", address, timeout)
	if err != nil {
		conn1.Close()
		return
	}

	cc1 := &MultiConn{}
	err = cc1.Init(nil, MultiConnParams{Connections: []net.Conn{conn1, conn2}})
	if err != nil {
		conn1.Close()
		conn2.Close()
		return
	}

	cc = cc1

	return
}

type InnerConnection struct {
	net.Conn
	ConnId  uint32
	isRead  bool
	isWrite bool
}

func (cc *InnerConnection) Read(b []byte) (n int, err error) {
	if !cc.isRead {
		err = binary.Read(cc.Conn, binary.BigEndian, cc.ConnId)
		if err != nil {
			return
		}

		cc.isRead = true
	}

	return cc.Conn.Read(b)
}

func (cc *InnerConnection) Write(b []byte) (n int, err error) {
	if !cc.isWrite {
		err = binary.Write(cc.Conn, binary.BigEndian, cc.ConnId)
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
	readItemsCh    chan ByteItem
	ReadReady      chan struct{}
	dataReadOffset uint32
	readBuffer     *bytes.Buffer
}

func NewBufferRead(DataReadOffset uint32) *DataBuffer {
	bs := new(DataBuffer)
	bs.readItemsCh = make(chan ByteItem, BufferItemsPerStream)
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
		item := pool.Get().(ByteItem)[:ByteItemLen]
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

type MultiConnectionArgs struct {
	ConnId uint32
}

type MultiConnection struct {
	Connections     map[int]connectionChannel
	connectionsLock sync.Mutex

	readItemsCh chan ByteItem

	ConnId         uint32
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

func (cc *MultiConnection) Init(args interface{}) error {
	if v, ok := args.(MultiConnectionArgs); ok {
		cc.ConnId = v.ConnId
		return nil
	}
	cc.readItemsCh = make(chan ByteItem, BufferItemsPerStream)
	cc.Connections = make(map[int]connectionChannel)
	return fmt.Errorf("arg error:%s", args)
}

func (cc *MultiConnection) Add(conn net.Conn) {
	cc.connectionsLock.Lock()
	defer cc.connectionsLock.Unlock()

	wrapConn := &InnerConnection{
		Conn:   conn,
		ConnId: cc.ConnId,
	}

	readBuffer := NewBufferRead(0)
	writeBuffer := NewBufferRead(0)

	connChannel := connectionChannel{
		Id:          time.Now().Nanosecond(),
		Conn:        wrapConn,
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
			return
		case <-r.ReadReady:
			item := pool.Get().(ByteItem)[:ByteItemLen]
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
		return
	}

}

func (cc *MultiConnection) Write(b []byte) (n int, err error) {
	data := b
	for len(data) > 0 {
		item := pool.Get().(ByteItem)[:ByteItemLen]
		nCopy := copy(item, data)
		n += nCopy
		item = item[:nCopy]
		data = data[nCopy:]

		//TODO: deal with write timeout
		for id, xx := range cc.Connections {
			_, err = xx.WriteBuffer.Write(item)
			if err != nil {
				xx.CancelFunc()
				delete(cc.Connections, id)
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
	return cc.Connections[0].LocalAddr()
}

func (cc *MultiConnection) RemoteAddr() net.Addr {
	return cc.Connections[0].RemoteAddr()
}

func (cc *MultiConnection) SetDeadline(t time.Time) error {

	for _, cc := range cc.Connections {
		err := cc.SetDeadline(t)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (cc *MultiConnection) SetReadDeadline(t time.Time) error {
	for _, cc := range cc.Connections {
		err := cc.SetReadDeadline(t)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (cc *MultiConnection) SetWriteDeadline(t time.Time) error {
	for _, cc := range cc.Connections {
		err := cc.SetWriteDeadline(t)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
