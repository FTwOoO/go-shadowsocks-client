package connection

import (
	"io"
	"net"
	"bytes"
	"errors"
)

var _ io.ReadWriteCloser = &DataBuffer{}

type DataBuffer struct {
	net.Conn
	readItemsCh     chan []byte
	readDataOffset  int
	writeDataOffset int
	readBuffer      *bytes.Buffer
	bufferItemLen   int
	closed          chan int
}

func NewBufferRead(bufferItemLen int, DataReadOffset int) *DataBuffer {
	bs := new(DataBuffer)

	maxBytes := 1024 * 64
	BufferItemsPerStream := (maxBytes / bufferItemLen)
	bs.readItemsCh = make(chan []byte, BufferItemsPerStream)
	bs.readBuffer = &bytes.Buffer{}
	bs.readDataOffset = DataReadOffset
	bs.bufferItemLen = bufferItemLen
	bs.closed = make(chan int)
	return bs
}

func (bs *DataBuffer) GetReadOffset() (n int) {
	return bs.readDataOffset
}

func (bs *DataBuffer) GetWriteOffset() (n int) {
	return bs.writeDataOffset
}

func (bs *DataBuffer) Write(b []byte) (n int, err error) {
	for len(b) > 0 {
		item := pool.Get().([]byte)
		item = item[0:bs.bufferItemLen]
		nCopy := copy(item, b)
		item = item[:nCopy]
		//TODO: set write timeout

		select {
		case <-bs.closed:
			err = errors.New("buffer closed")
			break
		case bs.readItemsCh <- item:
			b = b[nCopy:]
			n += nCopy
		}
	}

	if err != nil && n > 0 {
		bs.writeDataOffset += n
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
			bs.readDataOffset += nRead
			n += nRead
		} else {
			bs.readBuffer.Truncate(0)

			select {
			case <-bs.closed:
				err = errors.New("buffer closed")
				return
			case item, ok := <-bs.readItemsCh:
				if !ok {
					err = errors.New("readItemsCh closed")
					return
				}

				nCopy := copy(b, item)
				b = b[nCopy:]
				bs.readDataOffset += nCopy
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

func (bs *DataBuffer) Close() error {
	close(bs.closed)
	return nil
}


