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

type ClientConfig struct {
	ApplicationProtoConfig interface{}
	Detour                 bool
}

func StartClient(c *ClientConfig) context.CancelFunc {
	detour.InitSiteStat("stat.json")
	ctx, cancel := context.WithCancel(context.Background())

	transportDial := net.DialTimeout
	dial := c.ApplicationProtoConfig.(*dialer.SSPrococolConfig).GenClientDialer(transportDial)

	if c.Detour == true {
		dial = detour.GenDialer(dial, transportDial)
	}

	socksListenAddr, err := serv.SocksLocal(dial, ctx)
	if err != nil {
		panic(err)
	}
	proxy_setup.InitSocksProxySetting(socksListenAddr, ctx)
	return cancel
}

func main() {

	//systray.Run(onReady, onExit)

	var flags struct {
		Server   string
		Cipher   string
		Password string
	}

	flag.StringVar(&flags.Server, "server", "", "client connect address or url")
	flag.StringVar(&flags.Cipher, "cipher", "AEAD_CHACHA20_POLY1305", "available ciphers: "+strings.Join(core.ListCipher(), " "))
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.Parse()

	shadowsocks := &dialer.SSPrococolConfig{
		Cipher:     flags.Cipher,
		Password:   flags.Password,
		ServerAddr: flags.Server,
	}

	cancel := StartClient(&ClientConfig{
		ApplicationProtoConfig: shadowsocks,
		Detour: true,
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGIO, syscall.SIGABRT)
	signalMsg := <-quit
	log.Printf("signal[%v] received, ", signalMsg)
	cancel()

	//wait other goroutine to exit
	time.Sleep(3 * time.Second)
	log.Printf("program exit")
}
