package sitestat

import (
	"sync"
	"time"
)

// For once blocked site, use min dial/read timeout to make switching to
// parent proxy faster.
const minDialTimeout = 3 * time.Second
const minReadTimeout = 4 * time.Second
const defaultDialTimeout = 5 * time.Second
const defaultReadTimeout = 5 * time.Second
const maxTimeout = 15 * time.Second

var dialTimeout = defaultDialTimeout
var readTimeout = defaultReadTimeout

var siteStat = newSiteStat()
var storeLock sync.Mutex

//启动时调用
func InitSiteStat(sf string) {
	err := siteStat.load(sf)
	if err != nil {
		siteStat = newSiteStat()
		err = siteStat.load(sf + ".bak")
		if err != nil {
			siteStat = newSiteStat()
			siteStat.load("")
		}
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			storeLock.Lock()
			siteStat.store(sf)
			storeLock.Unlock()
		}
	}()
}
