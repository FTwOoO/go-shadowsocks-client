

shadowsocks client for macOS, based on go-shadowsocks2, inpired by flora-kit


## TODO
- local proxy setting
- local PAC service

## Build
```
make build
```

## Run


```
release/gss --cipher "AES-128-CFB" --password 123456 --c "server:9065" --socks ":7079"
```
