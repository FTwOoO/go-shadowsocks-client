package connection

import (
	"net"
	"reflect"
	"testing"
	"github.com/riobard/go-shadowsocks2/socks"
)

func TestShadowsocksRawConn_Read(t *testing.T) {
	testTarget := "google.com:443"

	paramsClient := ShadowsocksRawConnParams{Target: socks.ParseAddr(testTarget), IsServer: false}
	paramsServer := ShadowsocksRawConnParams{Target: nil, IsServer: true}

	l, _ := net.Listen("tcp", ":0")

	testBytes := []byte("aabbcc")

	go func() {
		cc3, _ := net.Dial("tcp", l.Addr().String())
		cc4 := &ShadowsocksRawConn{}
		cc4.Init(cc3, paramsClient)
		_, err := cc4.Write(testBytes)
		if err != nil {
			t.Fatal(err)
		}
	}()

	cc1, _ := l.Accept()
	cc2 := &ShadowsocksRawConn{}
	cc2.Init(cc1, paramsServer)

	b := make([]byte, len(testBytes))
	_, err := cc2.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(testBytes, b) {
		t.Fatalf("%s is not equal to %s", testBytes, b)
	}

	targetAddr := cc2.params.Target
	if targetAddr.String() != testTarget {
		t.Fatalf("%s != %s", targetAddr.String(), testTarget)
	}
}
