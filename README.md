# libsocks5connect

### install

```
go get golang.org/x/net/proxy
git clone https://github.com/ma6174/libsocks5connect.git && cd libsocks5connect
make
```

### use

```
export socks5_proxy=127.0.0.1:7070,localhost:1080
export not_proxy=127.0.0.0/8,192.168.1.0/24
export proxy_timeout_ms=1000
export proxy_no_log=false
LD_PRELOAD=./libsocks5connect.so curl google.com
```
