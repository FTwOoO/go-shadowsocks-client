package detour

import "strings"

type DetourStrategy int
const (
	AutoTry = DetourStrategy(1)
	AlwaysProxy = DetourStrategy(2)
	AlwaysDirect = DetourStrategy(3)
)


type DetourRules interface {
	 GetRule(host string) DetourStrategy
}


type CNRules struct {}
func (self *CNRules) GetRule(host string) DetourStrategy {
	if strings.HasSuffix(host, ".cn") {
		return AlwaysDirect
	}

	return AutoTry
}