
shadowsocks client for macOS, based on [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2), 
inspired by [flora-kit](https://github.com/huacnlee/flora-kit), [lantern](https://github.com/getlantern) and [cow](https://github.com/cyfdecyf/cow).

Auto set the system socks proxy, auto identify blocked sites, no config!



## Build
```
make build
```
or [download](https://github.com/FTwOoO/go-shadowsocks-client/files/1799215/gsc.zip) 

## Run
Server:
```
gss --cipher "AES-128-CFB" --password <password> --server <server>:<port>
```

Client:
```
gsc --cipher "AES-128-CFB" --password <password> --server <server>:<port>
```

(more ciphers are avalible, see [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2))

## TODO 
- systray
- auto select the fast proxy
