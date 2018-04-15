package main

import (
	"testing"
	"github.com/FTwOoO/go-ss/dialer"
	"github.com/FTwOoO/go-ss/serv"
	"context"
	"net"
	"golang.org/x/net/proxy"

	"fmt"
	"os"
	"net/http"
	"io/ioutil"
	"strings"
)

func TestServerAndClient(t *testing.T) {

	serverListenAddr := "127.0.0.1:15689"
	ssCipher := "AES-128-CFB"
	ssPswd := "12345678"


	//start server proxy
	shadowsocks := &dialer.SSPrococolConfig{
		Cipher:     ssCipher,
		Password:   ssPswd,
		ServerAddr: serverListenAddr,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := serv.TcpRemote(serverListenAddr, shadowsocks.GenServerConn, ctx)
	if err != nil {
		t.Fatal(err)
	}

	//start socks client
	//always detour
	dial := shadowsocks.GenClientDialer(net.DialTimeout)
	socksListenAddr, err := serv.SocksLocal(dial, ctx)
	if err != nil {
		t.Fatal(err)
	}

	dd, err := proxy.SOCKS5("tcp", socksListenAddr, nil, proxy.Direct)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't connect to the proxy:", err)
		os.Exit(1)
	}


	testURL := "http://example.com"
	httpClient := &http.Client{Transport: &http.Transport{Dial:dd.Dial}}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't create request:", err)
		os.Exit(2)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf( "can't GET page:%s", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf( "status code :%d", resp.StatusCode)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf( "error reading body:", err)
	}

	if !strings.Contains(string(b), "Example Domain") {
		t.Fatalf("Unexpected content:%s", string(b))
	} else {
		fmt.Println(string(b))
	}
}