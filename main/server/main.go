package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"github.com/riobard/go-shadowsocks2/core"
	"context"
	"github.com/FTwOoO/go-ss/dialer"
	"github.com/FTwOoO/go-ss/serv"
	"github.com/FTwOoO/go-ss/detour"
	"time"
)


func main() {
	ctx, cancel := context.WithCancel(context.Background())
	detour.InitSiteStat("stat.json")

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


	shadowsocks := &dialer.SSPrococolConfig{
		Cipher: flags.Cipher,
		Password:flags.Password,
	}

	err := serv.TcpRemote(flags.Server, shadowsocks.GenServerConn, ctx)
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

