package main

import (
	"encoding/base64"
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
	"github.com/FTwOoO/go-shadowsocks-client/serv"
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

	var key []byte
	if flags.Key != "" {
		k, err := base64.URLEncoding.DecodeString(flags.Key)
		if err != nil {
			log.Fatal(err)
		}
		key = k
	}


	addr := flags.Client
	cipher := flags.Cipher
	password := flags.Password
	var err error

	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Fatal(err)
	}

	proxy_setup.InitSocksProxySetting(flags.Socks, ctx)
	go serv.SocksLocal(flags.Socks, addr, ciph.StreamConn)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
}
