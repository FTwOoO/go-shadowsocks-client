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
	"github.com/FTwOoO/proxycore/proxy_setup"
	"github.com/FTwOoO/go-ss/detour"
	"github.com/FTwOoO/go-ss/dialer/protocol"
	_ "net/http/pprof"
	"net/http"
)

type ClientConfig struct {
	*protocol.SSProxyPrococol
	Detour bool
}

func StartClient(c *ClientConfig) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	detour.InitSiteStat("stat.json", ctx)

	var dial = net.DialTimeout
	dial = c.SSProxyPrococol.ClientWrapDial(net.DialTimeout)

	if c.Detour == true {
		/*		kcpDial := func (network, address string, timeout time.Duration) (net.Conn, error) {
					return kcp.Dial(address)
				}*/
		dial = detour.GenDial(dial, net.DialTimeout)
	}

	socksListenAddr, err := protocol.SocksServer(dial, ctx)
	if err != nil {
		panic(err)
	}
	proxy_setup.InitSocksProxySetting(socksListenAddr, ctx)
	return cancel
}

func main() {

	//systray.Run(onReady, onExit)

	var flags struct {
		Detour bool
		Server   string
		Cipher   string
		Password string
	}

	flag.BoolVar(&flags.Detour, "detour", false, "client connect address or url")
	flag.StringVar(&flags.Server, "server", "", "client connect address or url")
	flag.StringVar(&flags.Cipher, "cipher", "AEAD_CHACHA20_POLY1305", "available ciphers: "+strings.Join(core.ListCipher(), " "))
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.Parse()

	shadowsocks := &protocol.SSProxyPrococol{
		Cipher:     flags.Cipher,
		Password:   flags.Password,
		ServerAddr: flags.Server,
	}

	//fmt.Printf("detour flag: %v", flags.Detour)
	//os.Exit(1)
	cancel := StartClient(&ClientConfig{
		SSProxyPrococol: shadowsocks,
		Detour:          flags.Detour,
	})

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGIO, syscall.SIGABRT)
	signalMsg := <-quit
	log.Printf("signal[%v] received, ", signalMsg)
	cancel()

	//wait other goroutine to exit
	time.Sleep(3 * time.Second)
	log.Printf("program exit")
}
