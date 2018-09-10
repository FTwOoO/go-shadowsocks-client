package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"net"
	"context"
	"github.com/riobard/go-shadowsocks2/core"
	"github.com/FTwOoO/go-ss/serv"
	"github.com/FTwOoO/go-ss/dialer/protocol"
	"github.com/FTwOoO/kcp-go"
)

type ClientConfig struct {
	*protocol.SSProxyPrococol
	Detour bool
	UseKcp bool
}

func StartClient(c *ClientConfig) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	tcpDial := net.DialTimeout
	kcpDial := func(network, address string, timeout time.Duration) (net.Conn, error) {
		return kcp.Dial(address)
	}

	var dial = tcpDial

	if c.UseKcp {
		dial = kcpDial
	}

	proxyDial := c.SSProxyPrococol.ClientWrapDial(dial)

	if c.Detour == true { //deprecated
		//proxyDial = detour.GenDial(proxyDial, net.DialTimeout)
	}

	_, err := serv.SocksLocal(c.ListenAddr, proxyDial, ctx)
	if err != nil {
		panic(err)
	}
	//proxy_setup.InitSocksProxySetting(socksListenAddr, ctx)
	return cancel
}

func main() {

	//systray.Run(onReady, onExit)

	var flags struct {
		Server     string
		Cipher     string
		ListenAddr string
		Password   string
	}

	flag.StringVar(&flags.Server, "server", "", "client connect address or url")
	flag.StringVar(&flags.ListenAddr, "listen", "", "client connect address or url")
	flag.StringVar(&flags.Cipher, "cipher", "AEAD_CHACHA20_POLY1305", "available ciphers: "+strings.Join(core.ListCipher(), " "))
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.Parse()

	shadowsocks := &protocol.SSProxyPrococol{
		Cipher:     flags.Cipher,
		Password:   flags.Password,
		ServerAddr: flags.Server,
		ListenAddr: flags.ListenAddr,
	}

	cancel := StartClient(&ClientConfig{
		SSProxyPrococol: shadowsocks,
		Detour:          true,
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
