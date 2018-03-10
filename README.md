

shadowsocks client for macOS, based on [go-shadowsocks2](https://github.com/riobard/go-shadowsocks2), 
inpired by [flora-kit](https://github.com/huacnlee/flora-kit), [lantern](https://github.com/getlantern) and [cow](https://github.com/cyfdecyf/cow)


## TODO
- proxy setting
- local PAC service
- systray
- automatically identify blocked sites
- auto select the fast proxy

## Build
```
make build
```

## Run


```
release/gss --cipher "AES-128-CFB" --password 123456 --c "server:9065" --socks ":7079"
```
