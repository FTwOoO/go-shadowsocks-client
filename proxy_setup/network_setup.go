package proxy_setup

import (
	"os"
	"log"
	"runtime"
	"time"
	"context"
)

type SystemProxySettings interface {
	TurnOnGlobProxy()
	TurnOffGlobProxy()
}


func resetProxySettings(proxySettings SystemProxySettings, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Print("shutdown now ...")
			if nil != proxySettings{
				proxySettings.TurnOffGlobProxy()
			}
			time.Sleep(time.Duration(2000))
			os.Exit(0)
		}
	}
}

func InitProxySettings(bypass []string, addr string, ctx context.Context)  {
	var proxySettings SystemProxySettings
	if runtime.GOOS == "windows" {
		w := &windows{addr}
		proxySettings = w
	} else if runtime.GOOS == "darwin" {
		d := &darwin{bypass,addr}
		proxySettings = d
	}
	if nil != proxySettings{
		proxySettings.TurnOnGlobProxy()
	}
	go resetProxySettings(proxySettings, ctx)
}
