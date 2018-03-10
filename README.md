
shadowsocks client for macOS, based on [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2), 
inspired by [flora-kit](https://github.com/huacnlee/flora-kit), [lantern](https://github.com/getlantern) and [cow](https://github.com/cyfdecyf/cow).

Auto set the system socks proxy, auto identify blocked sites, no config!



## Build
```
go get -u golang.org/x/crypto/hkdf
go get -u github.com/riobard/go-shadowsocks2
go get -u github.com/getlantern/systray
go get -u github.com/stretchr/testify
make build
```

## Run

I recomment to use [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2) as server, run it:

```
go-shadowsocks2 -s ss://AES-128-CFB:<password>@:<port> -verbose
```

then run this client:

```
gsc --cipher "AES-128-CFB" --password <password> --c "<server>:<port>"
```

(more ciphers are avalible, see [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2))

## TODO 
- systray
- auto select the fast proxy