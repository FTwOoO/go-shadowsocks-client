package main

import (
	"context"
	"flag"
	"github.com/FTwOoO/go-ss/core"
	"github.com/FTwOoO/go-ss/dialer"
	"github.com/FTwOoO/go-ss/dialer/protocol"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var flags struct {
		Server   string
		Cipher   string
		Password string
		Socks    string
	}

	flag.StringVar(&flags.Server, "server", "", "server add to listen")
	flag.StringVar(&flags.Cipher, "cipher", "AEAD_CHACHA20_POLY1305", "available ciphers: "+strings.Join(core.ListCipher(), " "))
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.Parse()

	var shadowsocks dialer.ProxyProtocol = &protocol.SSProxyPrococol{
		Cipher:   flags.Cipher,
		Password: flags.Password,
	}

	err := shadowsocks.ServerListen(flags.Server, net.Listen, nil, ctx)
	if err != nil {
		panic(err)
	}

	/*

		kcpListen :=  func(net, laddr string) (net.Listener, error) {
			return kcp.Listen(laddr)
		}*/

	err = shadowsocks.ServerListen(flags.Server, net.Listen, nil, ctx)
	if err != nil {
		panic(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGIO, syscall.SIGABRT)
	signalMsg := <-quit
	log.Printf("signal[%v] received, ", signalMsg)
	cancel()

	//wait other goroutine to exit
	time.Sleep(3 * time.Second)
	log.Printf("program exit")
}
