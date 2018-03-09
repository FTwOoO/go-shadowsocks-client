package client

import (
	"net"
	"log"
	"github.com/FTwOoO/go-shadowsocks-client/proxy_setup"
	"context"
	"github.com/FTwOoO/go-shadowsocks-client/socks"
)


func Run(surgeCfg, geoipCfg string) {
	listenAddr := ""
	proxy_setup.InitProxySettings([]string{}, listenAddr, context.Background())
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listen socket", listenAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go socks.HandleSocksConnection(conn, nil)
	}
}