package connection

import (
	"testing"
	"net"
	"time"
	"crypto/rand"
	rand2 "math/rand"
	"fmt"
)

func TestNewMultiConnectionManager(t *testing.T) {
	EchoServer(t, "127.0.0.1:12345")
	EchoClient(t, "127.0.0.1:12345")
}

func EchoClient(t *testing.T, addr string) net.Conn {

	client := NewMultiConnectionManager()
	c, err := client.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Error(err)
	}

	for {
		n := rand2.Intn(1024)
		bytes := make([]byte, n)
		rand.Read(bytes)
		_, err := c.Write(bytes)
		if err != nil {
			t.Error(err)
		}

		fmt.Println("Echo write: %v", bytes)

		n2, err := c.Read(bytes)
		if err != nil {
			t.Error(err)
		}

		if n2 != n {
			t.Fatalf("Need %d bytes, get %d", n, n2)
		}

		fmt.Println("Echo read: %v", bytes)
	}

	return c

}

func EchoServer(t *testing.T, addr string) *MultiConnectionManager {
	server := NewMultiConnectionManager()
	l, err := server.Listen("tcp", addr)
	if err != nil {
		t.Error(err)
		return nil
	}

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				t.Error(err)
			}

			go func(c net.Conn) {
				body := make([]byte, 1024)
				n, err := c.Read(body)
				if err != nil {
					t.Error(err)
				}
				c.Write(body[:n])
				c.Close()
			}(c)
		}
	}()

	return server
}
