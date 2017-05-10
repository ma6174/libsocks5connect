# libsocks5connect

### install

```
go get golang.org/x/net/proxy
git clone https://github.com/ma6174/libsocks5connect.git && cd libsocks5connect
make
```

### config

```
export socks5_proxy=127.0.0.1:7070,localhost:1080
export not_proxy=127.0.0.0/8,192.168.1.0/24
export proxy_timeout_ms=1000
export proxy_no_log=false
```

### use

linux:

```
LD_PRELOAD=./libsocks5connect.so python -c 'import urllib2;print urllib2.urlopen("http://baidu.com").read()'
```

mac(experiential)

```
DYLD_FORCE_FLAT_NAMESPACE=1 DYLD_INSERT_LIBRARIES=./libsocks5connect.so python -c 'import urllib2;print urllib2.urlopen("http://baidu.com").read()'
```
