package detour

import (
	"net"
	"github.com/FTwOoO/go-shadowsocks-client/detour/sitestat"
)

type Detector struct {
	*sitestat.SiteStat
}

func (self *Detector) IsTimeout(err error) bool {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}

	return false
}

func (self *Detector) TryDirect(host string) bool {
	vcnt := self.SiteStat.GetVisitCnt(host)
	return (vcnt.AsDirect() || !vcnt.AsBlocked())
}

func (self *Detector) DirectVisit(host string) {
	vcnt := self.SiteStat.GetVisitCnt(host)
	vcnt.DirectVisit()
}


func (self *Detector) BlockedVisit(host string) {
	vcnt := self.SiteStat.GetVisitCnt(host)
	vcnt.BlockedVisit()
}

func (self *Detector) DontDirectVisit(host string) {
	vcnt := self.SiteStat.GetVisitCnt(host)
	vcnt.DontDirectVisit()
}

func (self *Detector) DontBlockedVisit(host string) {
	vcnt := self.SiteStat.GetVisitCnt(host)
	vcnt.DontBlockedVisit()
}