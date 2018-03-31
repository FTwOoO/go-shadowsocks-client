package dialer

import (
	"testing"
	"net"
	"reflect"
)

func TestCipherConn_Read(t *testing.T) {
	params := CipherConnParams{Cipher: "AES-128-CFB", Password: "123456"}

	l, _ := net.Listen("tcp", ":0")

	testBytes := []byte("aabbcc")

	go func () {
		cc3, _ := net.Dial("tcp", l.Addr().String())
		cc4 := &CipherConn{}
		cc4.Init(cc3, params)
		cc4.Write(testBytes)
	}()

	cc1, _ := l.Accept()
	cc2 := &CipherConn{}
	cc2.Init(cc1, params)

	b := make([]byte, len(testBytes))
	_, err := cc2.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(testBytes, b) {
		t.Fatalf("%s is not equal to %s", testBytes, b)
	}
}
