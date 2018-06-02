package connection

import (
	"net"
	"github.com/FTwOoO/go-ss/dialer"
	"sync"
	"fmt"
	"log"
	"time"
	"context"
	"bytes"
	"io"
	"github.com/FTwOoO/kcp-go"
	"github.com/getlantern/errors"
)

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

const ByteItemLen = 1024
const BufferItemsPerStream = 10

var pool = &sync.Pool{}

func init() {
	pool.New = func() interface{} {
		return make([]byte, ByteItemLen)
	}
}

type MultiConnParams struct {
	Connections []net.Conn
}

type ByteItem []byte

var _ io.ReadWriter = &ByteStream{}

type ByteStream struct {
	ch         chan ByteItem
	Offset     int
	readBuffer *bytes.Buffer
}

func NewByteStream() *ByteStream {
	c := new(ByteStream)
	c.ch = make(chan ByteItem, BufferItemsPerStream)
	c.readBuffer = &bytes.Buffer{}
	return c
}
func (bs *ByteStream) Write(b []byte) (n int, err error) {

	for len(b) > 0 {
		item := pool.Get().([]byte)
		nCopy := copy(item, b)
		item = item[:nCopy]
		n += nCopy
		bs.WriteItem(item)
	}
	return
}

func (bs *ByteStream) WriteItem(b ByteItem) {
	bs.ch <- b
}

func (bs *ByteStream) Read(b []byte) (n int, err error) {
	for {
		if bs.readBuffer.Len() > 0 {
			return bs.readBuffer.Read(b)
		}

		bs.readBuffer.Truncate(0)

		item := bs.ReadItem(1 * time.Second)
		if item == nil {
			continue
		}

		defer pool.Put(item)

		n = copy(b, item)
		if len(item) > n {
			bs.readBuffer.Write(item[n:])
		}
		return

	}
}

func (bs *ByteStream) ReadItem(timeout time.Duration) ByteItem {
	select {
	case c := <-bs.ch:
		bs.Offset += len(c)
		return c
	case <- time.After(timeout):
		return nil
	}
}

type MergeReadStream struct {
	*ByteStream
	ReadFrom []*ByteStream
}

func NewMergeReadStream(ReadFrom []*ByteStream, ctx context.Context) *MergeReadStream {
	c := new(MergeReadStream)
	c.ByteStream = NewByteStream()
	c.ReadFrom = ReadFrom

	go c.readLoop(ctx)
	return c
}

func (bs *MergeReadStream) readLoop(ctx context.Context) {

	for {
		for _, ibs := range bs.ReadFrom {
			item := ibs.ReadItem(500 * time.Millisecond)
			if item != nil {
				if bs.Offset >= ibs.Offset && bs.Offset < (ibs.Offset+len(item)) {
					ibs.Offset += len(item)
					item = item[bs.Offset-ibs.Offset:]
					bs.ByteStream.Write(item)
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	}
}

func (bs *MergeReadStream) WriteItem(b ByteItem) {
	panic("Not supported")
}

type MultiWriteStream struct {
	*ByteStream
	WriteTo []*ByteStream
}

func NewMultiWriteStream(WriteTo []*ByteStream, ctx context.Context) *MultiWriteStream {
	c := new(MultiWriteStream)
	c.ByteStream = NewByteStream()
	c.WriteTo = WriteTo

	go c.writeLoop(ctx)
	return c
}

func (bs *MultiWriteStream) writeLoop(ctx context.Context) {

	for {
		item := bs.ByteStream.ReadItem(500 * time.Millisecond)
		for _, ibs := range bs.WriteTo {
			ibs.Write(item)

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	}
}

func (bs *MultiWriteStream) ReadItem(timeout time.Duration) ByteItem {
	panic("Not supported")
}

var _ dialer.CommonConnection = &MultiConn{}

type MultiConn struct {
	params MultiConnParams

	readStream  []*ByteStream
	writeStream []*ByteStream

	mergeReadStream  *MergeReadStream
	multiWriteStream *MultiWriteStream

	items *sync.Pool

	readBuffer *bytes.Buffer
}

func (cc *MultiConn) Init(_ net.Conn, args interface{}) (err error) {
	if v, ok := args.(MultiConnParams); ok {
		cc.params = v

		cc.items = &sync.Pool{}
		cc.items.New = func() interface{} {
			return make([]byte, ByteItemLen)
		}

		for range cc.params.Connections {
			cc.readStream = append(cc.readStream, NewByteStream())
			cc.writeStream = append(cc.writeStream, NewByteStream())

		}
		cc.mergeReadStream = NewMergeReadStream(cc.readStream, context.Background())
		cc.multiWriteStream = NewMultiWriteStream(cc.writeStream, context.Background())
		cc.readBuffer = &bytes.Buffer{}
		return nil
	}

	return fmt.Errorf("args is not MultiConnParams:%s", args)
}

func (cc *MultiConn) ReadLoop() {
	for i, conn := range cc.params.Connections {
		go io.Copy(cc.readStream[i], conn)
		go io.Copy(conn, cc.writeStream[i])
	}

	go io.Copy(cc.multiWriteStream, cc.mergeReadStream)
}

func (cc *MultiConn) Read(b []byte) (n int, err error) {
	return cc.mergeReadStream.Read(b)
}

func (cc *MultiConn) Write(b []byte) (n int, err error) {
	return cc.multiWriteStream.Write(b)
}

func (cc *MultiConn) Close() error {
	for _, cc := range cc.params.Connections {
		err := cc.Close()
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (cc *MultiConn) LocalAddr() net.Addr {
	return cc.params.Connections[0].LocalAddr()
}

func (cc *MultiConn) RemoteAddr() net.Addr {
	return cc.params.Connections[0].RemoteAddr()
}

func (cc *MultiConn) SetDeadline(t time.Time) error {

	for _, cc := range cc.params.Connections {
		err := cc.SetDeadline(t)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (cc *MultiConn) SetReadDeadline(t time.Time) error {
	for _, cc := range cc.params.Connections {
		err := cc.SetReadDeadline(t)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (cc *MultiConn) SetWriteDeadline(t time.Time) error {
	for _, cc := range cc.params.Connections {
		err := cc.SetWriteDeadline(t)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
