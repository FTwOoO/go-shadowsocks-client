
shadowsocks client for macOS, based on [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2), 
inspired by [flora-kit](https://github.com/huacnlee/flora-kit), [lantern](https://github.com/getlantern) and [cow](https://github.com/cyfdecyf/cow).

* **not ready** Auto set the system socks proxy, auto identify blocked sites, no config!
* **not ready** raw KCP over multiple TCP connections

## Build
```
make build
```
or [download](https://github.com/FTwOoO/go-shadowsocks-client/files/1799215/gsc.zip) 

## Run
Server:
```
gsc --cipher "AES-128-CFB" --password <password> --server  "0.0.0.0:<port>" 
```

Client:

```
gsc --cipher "AES-128-CFB" --password <password> --server <proxy_server_ip>:<port> --listen "127.0.0.1:1080"
```

use with Chrome PLUGIN SwitchyOmega（AUTO PROXY MODE） 
