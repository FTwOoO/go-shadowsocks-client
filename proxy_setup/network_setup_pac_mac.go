package proxy_setup

var _ SystemProxySettings = &DarwinPac{}

type DarwinPac struct {
	PacUrl string
}

func (d *DarwinPac) TurnOffProxy() {
	execNetworks(func(deviceName string) {
		runNetworksetup("-setftpproxystate", deviceName, "off")
		runNetworksetup("-setwebproxystate", deviceName, "off")
		runNetworksetup("-setsecurewebproxystate", deviceName, "off")
		runNetworksetup("-setstreamingproxystate", deviceName, "off")
		runNetworksetup("-setgopherproxystate", deviceName, "off")
		runNetworksetup("-setsocksfirewallproxystate", deviceName, "off")
		runNetworksetup("-setproxyautodiscovery", deviceName, "off")
		runNetworksetup("-setautoproxystate", deviceName, "off")

	})
}

func (d *DarwinPac) TurnOnProxy() {
	execNetworks(func(deviceName string) {
		runNetworksetup("-setautoproxyurl", deviceName, d.PacUrl)
	})

	execNetworks(func(deviceName string) {
		runNetworksetup("-setautoproxystate", deviceName, "off")
	})
}
