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
	"github.com/FTwOoO/go-shadowsocks-client/proxy_setup"
	"github.com/getlantern/systray/example/icon"
	"github.com/getlantern/systray"
	"fmt"
	"github.com/FTwOoO/go-shadowsocks-client/dialer"
	"github.com/FTwOoO/go-shadowsocks-client/serv"
	"time"
	"net"
)

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("gss")
	systray.SetTooltip("a shadowsocks client")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	for {
		select {
		case <-mQuit.ClickedCh:
			systray.Quit()
			fmt.Println("Quit2 now...")
			return
		}
	}
}

func onExit() {
	// clean up here
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	//systray.Run(onReady, onExit)

	var flags struct {
		Client   string
		Cipher   string
		Key      string
		Password string
		Socks    string
	}

	flag.StringVar(&flags.Socks, "socks", "", "(client-only) SOCKS listen address")
	flag.StringVar(&flags.Client, "c", "", "client connect address or url")
	flag.StringVar(&flags.Cipher, "cipher", "AEAD_CHACHA20_POLY1305", "available ciphers: "+strings.Join(core.ListCipher(), " "))
	flag.StringVar(&flags.Key, "key", "", "base64url-encoded key (derive from password if empty)")
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.Parse()


	ssDialer := &dialer.ShadowsocksDialer{
		Cipher: flags.Cipher,
		Password:flags.Password,
		ServerAddr:flags.Client,
		Key: flags.Key,
	}


	var dial dialer.DialFunc = net.Dial
	dial = ssDialer.GenDialer(dial)


	proxy_setup.InitSocksProxySetting(flags.Socks, ctx)
	go serv.SocksLocal(flags.Socks, dial)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGIO, syscall.SIGABRT)
	signalMsg := <-quit
	log.Printf("signal[%v] received, ", signalMsg)
	cancel()

	//wait other goroutine to exit
	time.Sleep(3 * time.Second)
	log.Printf("Server shutdown completed, program exit")
}
