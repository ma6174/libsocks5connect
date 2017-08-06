package main

/*
#cgo LDFLAGS: -ldl -s -w
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
	if err == nil {
		return 0
	}
	if errno, ok := err.(syscall.Errno); ok {
		return C.int(errno)
	}
	return C.int(-1)
}

//export connect_proxy
func connect_proxy(fdc C.int, addr *C.struct_sockaddr, sockLen C.socklen_t) (ret C.int) {
	var (
		ip       []byte
		port     int
		sockAddr syscall.Sockaddr
		fd       = int(fdc)
	)
	var dialAddr string
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
		dialAddr = net.IP(ip).String() + ":" + fmt.Sprint(port)
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
		dialAddr = net.IP(ip).String() + ":" + fmt.Sprint(port)
	// case syscall.AF_UNIX:
	// addrLocal := (*syscall.RawSockaddrUnix)(unsafe.Pointer(addr))
	// var b []byte
	// for _, v := range addrLocal.Path {
	// if v == 0 {
	// break
	// }
	// b = append(b, byte(v))
	// }
	// dialAddr = fmt.Sprintf("%v", string(b))
	default:
		return C.orig_connect(fdc, addr, sockLen)
	}
	err := syscall.SetNonblock(fd, false)
	if err != nil {
		log.Printf("[fd:%v] syscall.SetNonblock failed: %v", fd, err)
		return C.setErrno(errno(err))
	}
	opt, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TYPE)
	if err != nil {
		log.Printf("[fd:%v] syscall.GetsockoptInt failed: %v", fd, err)
		return C.setErrno(errno(err))
	}
	var errCh = make(chan error, 1)
	var proxyUsed *ProxyAddr
	conn := NewFdConn(fd)
	defer conn.Close()
	if opt != syscall.SOCK_STREAM || config.GetProxyCount() == 0 || config.ShouldNotProxy(net.IP(ip)) || sockAddr == nil {
		go func() {
			ret := C.orig_connect(fdc, addr, sockLen)
			if ret == 0 {
				log.Printf("[fd:%v] direct connect success: %v -> %v", fd, conn.LocalAddr(), dialAddr)
				errCh <- nil
				return
			}
			err := syscall.Errno(ret)
			errCh <- err
		}()
	} else {
		proxyUsed = config.GetProxyAddr()
		go func() {
			err := syscall.Connect(fd, proxyUsed.Sockaddr())
			if err != nil {
				log.Printf("[fd:%v] syscall.Connect failed: %v", fd, err)
				errCh <- err
				return
			}
			dialer, err := proxy.SOCKS5("", "", &proxyUsed.Auth, conn)
			if err != nil {
				log.Printf("[fd:%v] proxy.SOCKS5 failed: %v", fd, err)
				errCh <- err
				return
			}
			_, err = dialer.Dial("tcp", dialAddr)
			if err != nil {
				log.Printf("[fd:%v] dialer Dial %v failed: %v", fd, dialAddr, err)
				errCh <- err
				return
			}
			log.Printf("[fd:%v] proxy connect success: %v -> %v -> %v", fd, conn.LocalAddr(), proxyUsed, dialAddr)
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
			log.Printf("[fd:%v] direct connect to %v failed %v", fd, dialAddr, err)
		} else {
			log.Printf("[fd:%v] connect to %v using proxy %v failed: %v",
				fd, dialAddr, proxyUsed, err)
		}
		return C.setErrno(errno(err))
	}
	return 0
}

//export close
func close(fdc C.int) C.int {
	fd := int(fdc)
	if opt, _ := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TYPE); opt == syscall.SOCK_STREAM {
		log.Printf("[fd:%v] close conn %v", fd, NewFdConn(fd).LocalAddr())
	}
	return C.setErrno(errno(syscall.Close(fd)))
}
