package detour

import (
	"net"
	"github.com/FTwOoO/go-ss/detour/sitestat"
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

	ip:=net.ParseIP(host)
	if ip != nil {
		return true
	}

	vcnt := self.SiteStat.GetVisitCnt(host)
	return (vcnt.AsDirect() || !vcnt.AsBlocked())
}

func (self *Detector) DirectVisitSuccess(host string) {
	vcnt := self.SiteStat.GetVisitCnt(host)
	vcnt.DirectVisit()
}


func (self *Detector) BlockedVisitSuccess(host string) {
	vcnt := self.SiteStat.GetVisitCnt(host)
	vcnt.BlockedVisit()
}

