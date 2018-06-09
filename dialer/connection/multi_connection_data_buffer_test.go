package connection

import (
	"testing"
	"math/rand"
)

func TestDataBuffer_ReadWrite(t *testing.T) {

	itemLen := 1024

	bf := NewBufferRead(itemLen, 0)

	for i := 1; i <= itemLen*64; i++ {
		b := make([]byte, i)
		rand.Read(b)

		n, err := bf.Write(b)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(b) {
			t.Fatalf("Write %d bytes fail: %d success", len(b), n)
		}

		t.Logf("Write %d bytes", n)

		br := make([]byte, i)
		n, err = bf.Read(br)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(br) {
			t.Fatalf("Read %d bytes fail: %d bytes success", len(br), n)
		}

		t.Logf("Read %d bytes", n)

	}

}
