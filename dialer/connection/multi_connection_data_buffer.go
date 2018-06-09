package connection

import (
	"io"
	"net"
	"bytes"
	"errors"
)

var _ io.ReadWriter = &DataBuffer{}

type DataBuffer struct {
	net.Conn
	readItemsCh    chan []byte
	dataReadOffset int
	readBuffer     *bytes.Buffer
	bufferItemLen  int
}

func NewBufferRead(bufferItemLen int, DataReadOffset int) *DataBuffer {
	bs := new(DataBuffer)

	maxBytes := 1024 * 64
	BufferItemsPerStream := (maxBytes / bufferItemLen)
	bs.readItemsCh = make(chan []byte, BufferItemsPerStream)
	bs.readBuffer = &bytes.Buffer{}
	bs.dataReadOffset = DataReadOffset
	bs.bufferItemLen = bufferItemLen
	return bs
}

func (bs *DataBuffer) GetDataReadOffset() (n int) {
	return bs.dataReadOffset
}

func (bs *DataBuffer) Write(b []byte) (n int, err error) {
	for len(b) > 0 {
		item := pool.Get().([]byte)
		item = item[0:bs.bufferItemLen]
		nCopy := copy(item, b)
		item = item[:nCopy]
		n += nCopy
		//TODO: set write timeout
		bs.readItemsCh <- item
		b = b[nCopy:]
	}
	return
}

func (bs *DataBuffer) Read(b []byte) (n int, err error) {

	for len(b) > 0 {
		if bs.readBuffer.Len() > 0 {
			nRead, err2 := bs.readBuffer.Read(b)
			if err2 != nil {
				err = err2
				return
			}
			b = b[nRead:]
			bs.dataReadOffset += nRead
			n += nRead
		} else {
			bs.readBuffer.Truncate(0)

			select {
			case item, ok := <-bs.readItemsCh:
				if !ok {
					err = errors.New("readItemsCh closed")
					return
				}

				nCopy := copy(b, item)
				b = b[nCopy:]
				bs.dataReadOffset += nCopy
				n += nCopy

				if len(item) > nCopy {
					bs.readBuffer.Write(item[nCopy:])
				}
				pool.Put(item)

			default:
				return
			}
		}
	}

	return
}
