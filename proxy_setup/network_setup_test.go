// +build darwin
// +build amd64 386

package proxy_setup

import (
	"testing"
)


func TestListNetworks(t *testing.T) {
	nss := listNetworks()

	for _, v := range nss {
		t.Log("|" + v + "|")
	}
}