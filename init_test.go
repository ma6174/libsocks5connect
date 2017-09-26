package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	r := require.New(t)
	config.proxyAddrs = nil
	r.True(strings.Contains(config.String(), "proxy_timeout_ms=1000"))
	conn, err := net.Dial("tcp", configAddr)
	r.NoError(err)
	go io.Copy(os.Stdout, conn)

	_, err = fmt.Fprintln(conn, "proxy_timeout_ms=2000")
	r.NoError(err)
	time.Sleep(time.Millisecond * 10)
	r.True(strings.Contains(config.String(), "proxy_timeout_ms=2000"))
	r.Equal(2000*time.Millisecond, config.connectTimeouts)

	_, err = fmt.Fprintln(conn, "socks5_proxy=127.0.0.1:100,127.0.0.1:101")
	r.NoError(err)
	time.Sleep(time.Millisecond * 10)
	r.Equal("127.0.0.1:100", config.proxyAddrs[0].String())
	r.Equal("127.0.0.1:101", config.proxyAddrs[1].String())

	_, err = fmt.Fprintln(conn, "not_proxy=192.168.1.0/24")
	r.NoError(err)
	time.Sleep(time.Millisecond * 10)
	r.True(config.ShouldNotProxy(net.IP{192, 168, 1, 100}))
	r.False(config.ShouldNotProxy(net.IP{192, 168, 2, 100}))

	_, err = fmt.Fprintln(conn, "socks5_proxy=192.168.111.111:1000")
	r.NoError(err)
	time.Sleep(time.Millisecond * 10)
	r.Equal("192.168.111.111:1000", config.GetProxyAddr().String())
	_, err = fmt.Fprintln(conn, "socks5_proxy=192.168.111.112:2222")
	r.NoError(err)
	time.Sleep(time.Millisecond * 10)
	r.Equal("192.168.111.112:2222", config.GetProxyAddr().String())
	r.Equal([]string{"192.168.111.112:2222"}, config.GetProxyAddrs())
}
