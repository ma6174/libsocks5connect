# libsocks5connect

```
make
export socks5_proxy=127.0.0.1:7070,localhost:1080
export not_proxy=127.0.0.0/8,192.168.1.0/24
LD_PRELOAD=./libsocks5connect.so curl google.com
```
