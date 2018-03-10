
shadowsocks client for macOS, based on [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2), 
inspired by [flora-kit](https://github.com/huacnlee/flora-kit), [lantern](https://github.com/getlantern) and [cow](https://github.com/cyfdecyf/cow)


## Feature
- proxy setting
- automatically identify blocked sites

## Build
```
go get -u golang.org/x/crypto/hkdf
go get -u github.com/riobard/go-shadowsocks2
go get -u github.com/getlantern/systray
go get -u github.com/stretchr/testify
make build
```

## Run
```
release/gss --cipher "AES-128-CFB" --password <password> --c "<server>:<port>" --socks ":7079"
```

## TODO 
- local PAC service
- systray
- auto select the fast proxy