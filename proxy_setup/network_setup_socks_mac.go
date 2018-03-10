package proxy_setup

import (
	"net"
	"github.com/labstack/gommon/log"
)

var _ SystemProxySettings = &DarwinSocks{}

type DarwinSocks struct {
	bypassDomains []string
	address       string
}

func (d *DarwinSocks) TurnOffProxy() {
	execNetworks(func(deviceName string) {
		runNetworksetup("-setftpproxystate", deviceName, "off")
		runNetworksetup("-setwebproxystate", deviceName, "off")
		runNetworksetup("-setsecurewebproxystate", deviceName, "off")
		runNetworksetup("-setstreamingproxystate", deviceName, "off")
		runNetworksetup("-setgopherproxystate", deviceName, "off")
		runNetworksetup("-setsocksfirewallproxystate", deviceName, "off")
		runNetworksetup("-setproxyautodiscovery", deviceName, "off")
	})
}

func (d *DarwinSocks) TurnOnProxy() {
	host, port, _ := net.SplitHostPort(d.address)
	if host == "" {
		host = "127.0.0.1"
	}

	execNetworks(func(deviceName string) {
		log.Printf(runNetworksetup("-setsocksfirewallproxy", deviceName, host, port))
	})

	if len(d.bypassDomains) > 0 {
		execNetworks(func(deviceName string) {
			args := []string{"-setproxybypassdomains", deviceName}
			args = append(args, d.bypassDomains...)
			runNetworksetup(args...)
		})
	}
}
