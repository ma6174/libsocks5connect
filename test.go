// +build test

package main

/*
#include <sys/types.h>
#include <arpa/inet.h>
#include <errno.h>
static inline int setErrno(int err) {
	if (err == 0) {
		return 0;
	}
	errno = err;
	return -1;
}
*/
import "C"
import (
	"log"
	"net"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/armon/go-socks5"
	"github.com/stretchr/testify/require"
)

func testSocket(t *testing.T) {
	// listen
	r := require.New(t)
	fdListen, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	r.NoError(err)
	err = syscall.Bind(fdListen, &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{127, 0, 0, 1},
	})
	r.NoError(err)
	err = syscall.Listen(fdListen, 10)
	r.NoError(err)
	sa, err := syscall.Getsockname(fdListen)
	r.NoError(err)
	listenAddr := netAddr(sa).(*net.TCPAddr)
	r.NotNil(listenAddr)
	log.Println("listen:", listenAddr)
	// accept
	go func() {
		for {
			var (
				addr    C.struct_sockaddr
				sockLen C.socklen_t
			)
			ret := accept(C.int(fdListen), &addr, &sockLen)
			log.Println(ret, addr, sockLen)
			p := make([]byte, 1)
			_, err := syscall.Read(int(ret), p)
			r.NoError(err)
			_, err = syscall.Write(int(ret), p)
			r.NoError(err)
		}
	}()
	socks5Ln, err := net.Listen("tcp", "127.0.0.1:0")
	r.NoError(err)
	svr, err := socks5.New(&socks5.Config{})
	r.NoError(err)
	go svr.Serve(socks5Ln)
	proxyAddrs := []ProxyAddr{ProxyAddr{
		AddrStr:      socks5Ln.Addr().String(),
		ResolvedAddr: socks5Ln.Addr().(*net.TCPAddr),
	}}
	config.connectTimeouts = time.Second
	// proxy connect
	{
		for _, proxy := range [][]ProxyAddr{[]ProxyAddr{}, proxyAddrs} {
			config.proxyAddrs = proxy
			fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
			r.NoError(err)
			err = syscall.SetNonblock(fd, false)
			r.NoError(err)
			addr := (*C.struct_sockaddr)(unsafe.Pointer(&syscall.RawSockaddrInet4{
				Family: syscall.AF_INET,
				Addr:   [4]byte{127, 0, 0, 1},
				Port:   uint16(listenAddr.Port<<8 | listenAddr.Port>>8),
			}))
			ret := connect_proxy(C.int(fd), addr, C.socklen_t(8))
			r.Equal(0, int(ret))
			data := []byte("a")
			_, err = syscall.Write(fd, data)
			r.NoError(err)
			var out = make([]byte, 1)
			_, err = syscall.Read(fd, out)
			r.NoError(err)
			r.Equal(data, out)
			ret = close(C.int(fd))
			r.Equal(0, int(ret))
		}
	}
	// connect wrong port
	{
		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
		r.NoError(err)
		addr := (*C.struct_sockaddr)(unsafe.Pointer(&syscall.RawSockaddrInet4{
			Family: syscall.AF_INET,
			Addr:   [4]byte{127, 0, 0, 1},
			Port:   uint16(listenAddr.Port<<8|listenAddr.Port>>8) + 1,
		}))
		ret := connect_proxy(C.int(fd), addr, C.socklen_t(8))
		r.Equal(-1, int(ret))
		ret = close(C.int(fd))
		r.Equal(0, int(ret))
	}
}
