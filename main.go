package main

import (
	"github.com/FTwOoO/go-shadowsocks-client/client"
)



func main() {
	var configFile, geoipdb string
	configFile="/Users/ganxiangle/Workspaces/Mac-Setup/bin/gopath/src/github.com/huacnlee/client-kit/client.default.conf"
	geoipdb="/Users/ganxiangle/Workspaces/Mac-Setup/bin/gopath/src/github.com/huacnlee/client-kit/geoip.mmdb"
	client.Run(configFile, geoipdb)

}
