package main

/*
#cgo LDFLAGS: -ldl -s -w
#include <arpa/inet.h>
#include <errno.h>
static inline int setErrno(int err) {
     errno = err;
     return -1;
}
int orig_connect(int socket, const struct sockaddr *address, socklen_t address_len);
*/
import "C"
import (
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/net/proxy"
)

func errno(err error) C.int {
	if errno, ok := err.(syscall.Errno); ok {
		return C.int(errno)
	}
	return C.int(-1)
}

//export connect_proxy
func connect_proxy(fd C.int, addr *C.struct_sockaddr, sockLen C.socklen_t) (ret C.int) {
	var (
		ip       []byte
		port     int
		sockAddr syscall.Sockaddr
	)
	goAddr := (*syscall.RawSockaddr)(unsafe.Pointer(addr))
	switch goAddr.Family {
	case syscall.AF_INET:
		addr4 := (*syscall.RawSockaddrInet4)(unsafe.Pointer(addr))
		port = int(addr4.Port<<8 | addr4.Port>>8)
		ip = addr4.Addr[:]
		var ip4 [4]byte
		copy(ip4[:], ip)
		sockAddr = &syscall.SockaddrInet4{
			Addr: ip4,
			Port: port,
		}
	case syscall.AF_INET6:
		addr6 := (*syscall.RawSockaddrInet6)(unsafe.Pointer(addr))
		ip = addr6.Addr[:]
		port = int(addr6.Port<<8 | addr6.Port>>8)
		var ip6 [16]byte
		copy(ip6[:], ip)
		sockAddr = &syscall.SockaddrInet6{
			Addr:   ip6,
			Port:   port,
			ZoneId: addr6.Scope_id,
		}
	default:
		log.Printf("unknown addr Family: %+v, direct connect", goAddr)
	}
	dialAddr := net.IP(ip).String() + ":" + fmt.Sprint(port)
	if sockAddr == nil {
		dialAddr = fmt.Sprintf("%+v", addr)
	}
	err := syscall.SetNonblock(int(fd), false)
	if err != nil {
		log.Println("err", err)
		return C.setErrno(errno(err))
	}
	opt, err := syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_TYPE)
	if err != nil {
		log.Println("syscall.GetsockoptInt failed", err)
		return C.setErrno(errno(err))
	}
	var errCh = make(chan error, 1)
	var proxyUsed *ProxyAddr
	if opt == syscall.SOCK_DGRAM || config.GetProxyCount() == 0 || config.ShouldNotProxy(net.IP(ip)) || sockAddr == nil {
		go func() {
			ret = C.orig_connect(fd, addr, sockLen)
			if ret == 0 {
				log.Printf("direct connect to %v success", dialAddr)
				errCh <- nil
				return
			}
			err := syscall.Errno(ret)
			log.Println("connect failed", err)
			errCh <- err
		}()
	} else {
		proxyUsed = config.GetProxyAddr()
		conn := NewFdConn(int(fd))
		defer conn.Close()
		go func() {
			err := syscall.Connect(int(fd), proxyUsed.Sockaddr())
			if err != nil {
				log.Println("syscall.Connect failed", err)
				errCh <- err
				return
			}
			dialer, err := proxy.SOCKS5("", "", &proxyUsed.Auth, conn)
			if err != nil {
				log.Println("proxy.SOCKS5 failed", err)
				errCh <- err
				return
			}
			_, err = dialer.Dial("tcp", dialAddr)
			if err != nil {
				log.Printf("dialer Dial %v failed: %v", dialAddr, err)
				errCh <- err
				return
			}
			log.Printf("connect to %v using proxy %v success", dialAddr, proxyUsed)
			errCh <- nil
		}()
	}
	select {
	case <-time.After(config.GetConnectTimeouts()):
		err = syscall.ETIMEDOUT
	case err = <-errCh:
	}
	if err != nil {
		if proxyUsed == nil {
			log.Printf("direct connect to %v failed %v", dialAddr, err)
		} else {
			log.Printf("connect to %v using proxy %v failed: %v",
				dialAddr, proxyUsed, err)
		}
		return C.setErrno(errno(err))
	}
	return 0
}
