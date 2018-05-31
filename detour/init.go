package detour

import (
	"sync"
	"time"
	"github.com/FTwOoO/go-ss/detour/sitestat"
	"context"
	"log"
)

const minDialTimeout = 3 * time.Second
const minReadTimeout = 4 * time.Second
const defaultDialTimeout = 5 * time.Second
const defaultReadTimeout = 10 * time.Second
const maxTimeout = 15 * time.Second

var siteStat = sitestat.NewSiteStat()

//启动时调用
func InitSiteStat(sf string, ctx context.Context) {
	var storeLock sync.Mutex

	err := siteStat.Load(sf)
	if err != nil {
		log.Printf("Load data file fail: %s, %v\n", sf, err)
	}

	go func() {
		for {
			select {
			case <- time.After(30*time.Second):
				storeLock.Lock()
				siteStat.Store(sf)
				storeLock.Unlock()
			case <- ctx.Done():
				storeLock.Lock()
				siteStat.Store(sf)
				storeLock.Unlock()
				return
			}
		}
	}()
}
