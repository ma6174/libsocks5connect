# libsocks5connect

### install

```
git clone https://github.com/ma6174/libsocks5connect.git && cd libsocks5connect && make
```

### config

```
$ cat proxy.conf
export socks5_proxy=127.0.0.1:7070,localhost:1080
export not_proxy=127.0.0.0/8,192.168.1.0/24
export proxy_timeout_ms=1000
export proxy_no_log=false
export libsocks5connect=./libsocks5connect.so
```

### use

```
./proxy.sh -f proxy.conf python2 -c 'import urllib2;print(urllib2.urlopen("http://google.com").read())'
```
