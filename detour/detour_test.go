package detour

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	directMsg string = "hello direct"
	detourMsg string = "hello detour"
)

func proxyTo(proxiedURL string) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		u, _ := url.Parse(proxiedURL)
		return net.Dial("tcp", u.Host)
	}
}

func TestBlockedImmediately(t *testing.T) {
	defer stopMockServers()
	proxiedURL, _ := newMockServer(detourMsg)
	TimeoutToDetour = 50 * time.Millisecond
	mockURL, mock := newMockServer(directMsg)

	client := &http.Client{Timeout: 50 * time.Millisecond}
	mock.Timeout(200*time.Millisecond, directMsg)
	resp, err := client.Get(mockURL)
	assert.Error(t, err, "direct access to a timeout url should fail")

	client = newClient(proxiedURL, 100*time.Millisecond)
	resp, err = client.Get("http://255.0.0.1") // it's reserved for future use so will always time out
	if assert.NoError(t, err, "should have no error if dialing times out") {
		assert.True(t, wlTemporarily("255.0.0.1:80"), "should be added to whitelist if dialing times out")
		assertContent(t, resp, detourMsg, "should detour if dialing times out")
	}

	client = newClient(proxiedURL, 100*time.Millisecond)
	resp, err = client.Get("http://127.0.0.1:4325") // hopefully this port didn't open, so connection will be refused
	if assert.NoError(t, err, "should have no error if connection is refused") {
		assert.True(t, wlTemporarily("127.0.0.1:4325"), "should be added to whitelist if connection is refused")
		assertContent(t, resp, detourMsg, "should detour if connection is refused")
	}

	u, _ := url.Parse(mockURL)
	resp, err = client.Get(mockURL)
	if assert.NoError(t, err, "should have no error if reading times out") {
		assert.True(t, wlTemporarily(u.Host), "should be added to whitelist if reading times out")
		assertContent(t, resp, detourMsg, "should detour if reading times out")
	}

	client = newClient(proxiedURL, 100*time.Millisecond)
	RemoveFromWl(u.Host)
	resp, err = client.PostForm(mockURL, url.Values{"key": []string{"value"}})
	if assert.Error(t, err, "Non-idempotent method should not be detoured in same connection") {
		assert.True(t, wlTemporarily(u.Host), "but should be added to whitelist so will detour next time")
	}
}

func TestBlockedAfterwards(t *testing.T) {
	defer stopMockServers()
	proxiedURL, _ := newMockServer(detourMsg)
	TimeoutToDetour = 50 * time.Millisecond
	mockURL, mock := newMockServer(directMsg)
	client := newClient(proxiedURL, 100*time.Millisecond)

	mock.Msg(directMsg)
	resp, err := client.Get(mockURL)
	if assert.NoError(t, err, "should have no error for normal response") {
		assertContent(t, resp, directMsg, "should access directly for normal response")
	}
	mock.Timeout(200*time.Millisecond, directMsg)
	_, err = client.Get(mockURL)
	assert.Error(t, err, "should have error if reading times out for a previously worked url")
	resp, err = client.Get(mockURL)
	if assert.NoError(t, err, "but should have no error for the second time") {
		u, _ := url.Parse(mockURL)
		assert.True(t, wlTemporarily(u.Host), "should be added to whitelist if reading times out")
		assertContent(t, resp, detourMsg, "should detour if reading times out")
	}
}

func TestRemoveFromWhitelist(t *testing.T) {
	defer stopMockServers()
	proxiedURL, proxy := newMockServer(detourMsg)
	proxy.Timeout(200*time.Millisecond, detourMsg)
	TimeoutToDetour = 50 * time.Millisecond
	mockURL, _ := newMockServer(directMsg)
	client := newClient(proxiedURL, 100*time.Millisecond)

	u, _ := url.Parse(mockURL)
	AddToWl(u.Host, false)
	_, err := client.Get(mockURL)
	if assert.Error(t, err, "should have error if reading times out through detour") {
		time.Sleep(250 * time.Millisecond)
		assert.False(t, whitelisted(u.Host), "should be removed from whitelist if reading times out through detour")
	}

}

func TestClosing(t *testing.T) {
	defer stopMockServers()
	proxiedURL, proxy := newMockServer(detourMsg)
	proxy.Timeout(200*time.Millisecond, detourMsg)
	TimeoutToDetour = 50 * time.Millisecond
	mockURL, mock := newMockServer(directMsg)
	mock.Msg(directMsg)
	DirectAddrCh = make(chan string)
	{
		if _, err := newClient(proxiedURL, 100*time.Millisecond).Get(mockURL); err != nil {
			log.Debugf("Unable to send GET request to mock URL: %v", err)
		}
	}
	u, _ := url.Parse(mockURL)
	addr := <-DirectAddrCh
	assert.Equal(t, u.Host, addr, "should get notified when a direct connetion has no error while closing")
}

func newClient(proxyURL string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: Dialer(proxyTo(proxyURL))},
		Timeout: timeout,
	}
}
func assertContent(t *testing.T, resp *http.Response, msg string, reason string) {
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err, reason)
	assert.Equal(t, msg, string(b), reason)
}
