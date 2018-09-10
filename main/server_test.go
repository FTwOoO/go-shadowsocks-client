package main

import (
	"testing"
	"github.com/FTwOoO/go-ss/dialer"
	"context"
	"net"
	"golang.org/x/net/proxy"
	"net/http"
	"io/ioutil"
	"strings"
	"github.com/FTwOoO/go-ss/dialer/protocol"
)

func TestServerAndClient(t *testing.T) {

	serverListenAddr := "127.0.0.1:16923"
	ssCipher := "AES-128-CFB"
	ssPswd := "12345678"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//start server proxy

	var shadowsocks dialer.ProxyProtocol = &protocol.SSProxyPrococol{
		Cipher:     ssCipher,
		Password:   ssPswd,
		ServerAddr: serverListenAddr,
	}

	err := shadowsocks.ServerListen(serverListenAddr, net.Listen, nil, ctx)
	if err != nil {
		panic(err)
	}

	dial := shadowsocks.ClientWrapDial(net.DialTimeout)
	socksListenAddr, err := protocol.SocksServer("0.0.0.0:0", dial, ctx)
	if err != nil {
		t.Fatal(err)
	}

	dd, err := proxy.SOCKS5("tcp", socksListenAddr, nil, proxy.Direct)
	if err != nil {
		t.Fatalf("can't connect to the proxy:", err)
	}

	testURL := "http://example.com"
	httpClient := &http.Client{Transport: &http.Transport{Dial: dd.Dial}}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		t.Fatalf("can't create request:", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("can't GET page:%s", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status code :%d", resp.StatusCode)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading body:", err)
	}

	if !strings.Contains(string(b), "Example Domain") {
		t.Fatalf("Unexpected content:%s", string(b))
	}
}
