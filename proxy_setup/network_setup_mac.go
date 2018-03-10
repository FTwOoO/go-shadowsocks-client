package proxy_setup

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"regexp"
	"errors"
)

func runNetworksetup(args ...string) string {
	cmd := exec.Command("networksetup", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Println(err)
		log.Println(stderr.String())
	}
	return out.String()
}

type execNetworkFunc func(deviceName string)

func execNetworks(callback execNetworkFunc) {
	var allow_services = "Wi-Fi|Thunderbolt Bridge|Thunderbolt Ethernet"

	for _, deviceName := range listNetworks() {
		if !strings.Contains(allow_services, deviceName) {
			continue
		}
		callback(deviceName)
	}
}

func listNetworks() ([]string) {
	c := exec.Command("networksetup", "-listallnetworkservices")
	out, err := c.CombinedOutput()
	if err != nil {
		log.Println(errors.New("ns lans:" + string(out) + ":" + err.Error()))
		return nil
	}
	nss := make([]string, 0)
	reg := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	for _, v := range bytes.Split(bytes.TrimSpace(out), []byte("\n")) {
		// An asterisk (*) denotes that a network service is disabled.
		if bytes.Contains(v, []byte("*")) {
			continue
		}
		ns := string(bytes.TrimSpace(v))
		c := exec.Command("networksetup", "-getinfo", ns)
		out, err := c.CombinedOutput()
		if err != nil {
			log.Println(errors.New("ns gi:" + string(out) + ":" + err.Error()))
			return nil
		}
		if !reg.MatchString(string(out)) {
			continue
		}
		nss = append(nss, ns)
	}
	if len(nss) == 0 {
		log.Println(errors.New("no available network service"))

		return nil
	}
	return nss
}
