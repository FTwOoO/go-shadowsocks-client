package sitestat

import "testing"

func TestSiteStat_domainSuffix(t *testing.T) {

	if "eastday.com" != domainSuffix("09.imgmini.eastday.com") {
		t.FailNow()
	}

	if "rackcdn.com" != domainSuffix("575381e836a3d94dce07-544118806e68aa71eb4d3fce243a07c1.ssl.cf1.rackcdn.com") {
		t.FailNow()
	}
}