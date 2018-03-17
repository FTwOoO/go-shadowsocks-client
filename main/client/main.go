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
	"github.com/FTwOoO/proxycore/proxy_setup"
	"github.com/FTwOoO/go-ss/dialer"
	"github.com/FTwOoO/go-ss/serv"
	"time"
	"net"
	"github.com/FTwOoO/go-ss/detour"
)


func main() {
	ctx, cancel := context.WithCancel(context.Background())
	detour.InitSiteStat("stat.json")

	//systray.Run(onReady, onExit)

	var flags struct {
		Client   string
		Cipher   string
		Password string
	}

	flag.StringVar(&flags.Client, "s", "", "client connect address or url")
	flag.StringVar(&flags.Cipher, "cipher", "AEAD_CHACHA20_POLY1305", "available ciphers: "+strings.Join(core.ListCipher(), " "))
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.Parse()


	shadowsocks := &dialer.PrococolConfig{
		Cipher: flags.Cipher,
		Password:flags.Password,
		ServerAddr:flags.Client,
	}


	transportDial := net.DialTimeout
	applicationProtocolDial := shadowsocks.GenClientDialer(transportDial)
	dial := detour.GenDialer(applicationProtocolDial, transportDial)

	socksListenAddr, err := serv.SocksLocal(dial, ctx)
	if err != nil {
		panic(err)
	}
	proxy_setup.InitSocksProxySetting(socksListenAddr, ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGIO, syscall.SIGABRT)
	signalMsg := <-quit
	log.Printf("signal[%v] received, ", signalMsg)
	cancel()

	//wait other goroutine to exit
	time.Sleep(3 * time.Second)
	log.Printf("program exit")
}
