package proxy_setup

import (
	"log"
	"context"
)

type SystemProxySettings interface {
	TurnOffProxy()
	TurnOnProxy()
}

func InitSocksProxySetting(socksAddr string, ctx context.Context) {
	initProxySettings(&DarwinSocks{address: socksAddr}, ctx)
}

func initProxySettings(proxySettings SystemProxySettings, ctx context.Context) {
	proxySettings.TurnOnProxy()

	go func(proxySettings SystemProxySettings, ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				log.Print("Clean up system proxy setting  ...")
				if nil != proxySettings {
					proxySettings.TurnOffProxy()
				}
				return
			}
		}
	}(proxySettings, ctx)
}
