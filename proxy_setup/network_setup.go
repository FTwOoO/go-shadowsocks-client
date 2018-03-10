package proxy_setup

import (
	"os"
	"log"
	"time"
	"context"
)

type SystemProxySettings interface {
	TurnOffProxy()
	TurnOnProxy()
}

func InitSocksProxySetting(socksAddr string,  ctx context.Context)  {
	initProxySettings(&DarwinSocks{address:socksAddr}, ctx)
}

func initProxySettings(proxySettings SystemProxySettings, ctx context.Context) {
	proxySettings.TurnOnProxy()

	go func(proxySettings SystemProxySettings, ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				log.Print("shutdown now ...")
				if nil != proxySettings {
					proxySettings.TurnOffProxy()
				}
				time.Sleep(time.Duration(2000))
				os.Exit(0)
			}
		}
	}(proxySettings, ctx)
}
