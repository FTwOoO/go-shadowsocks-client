package detour

import (
	"sync"
	"time"
	"github.com/FTwOoO/go-shadowsocks-client/detour/sitestat"
)

const minDialTimeout = 3 * time.Second
const minReadTimeout = 4 * time.Second
const defaultDialTimeout = 3 * time.Second
const defaultReadTimeout = 5 * time.Second
const maxTimeout = 15 * time.Second

var siteStat = sitestat.NewSiteStat()

//启动时调用
func InitSiteStat(sf string) {
	var storeLock sync.Mutex

	err := siteStat.Load(sf)
	if err != nil {
		siteStat = sitestat.NewSiteStat()
		err = siteStat.Load(sf + ".bak")
		if err != nil {
			siteStat = sitestat.NewSiteStat()
			siteStat.Load("")
		}
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			storeLock.Lock()
			siteStat.Store(sf)
			storeLock.Unlock()
		}
	}()
}
