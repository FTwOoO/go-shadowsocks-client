package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"context"
	"time"
	"github.com/riobard/go-shadowsocks2/core"
	"github.com/FTwOoO/go-ss/serv"
	"github.com/FTwOoO/go-ss/dialer/protocol"
	"github.com/FTwOoO/go-ss/dialer"
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

	var shadowsocks dialer.ConnectionSpec = &protocol.SSProxyPrococol{
		Cipher:   flags.Cipher,
		Password: flags.Password,
	}

	err := serv.TcpRemote(flags.Server, shadowsocks.ServerWrapConn, ctx)
	if err != nil {
		panic(err)
	}


	err = serv.KcpRemote(flags.Server, shadowsocks.ServerWrapConn, ctx)
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
