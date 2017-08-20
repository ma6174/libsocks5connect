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
	fd := int(fdc)
	var dialAddr *net.TCPAddr
	var sockAddr syscall.Sockaddr
	goAddr := (*syscall.RawSockaddr)(unsafe.Pointer(addr))
	switch goAddr.Family {
	case syscall.AF_INET:
		addr4 := (*syscall.RawSockaddrInet4)(unsafe.Pointer(addr))
		dialAddr = &net.TCPAddr{
			IP:   addr4.Addr[:],
			Port: int(addr4.Port<<8 | addr4.Port>>8),
		}
		sockAddr = &syscall.SockaddrInet4{
			Addr: addr4.Addr,
			Port: int(addr4.Port<<8 | addr4.Port>>8),
		}
	default:
		log.Printf("[%d] syscall connect %+v", fd, addr)
		_, _, ret := syscall.Syscall(syscall.SYS_CONNECT, uintptr(fdc),
			uintptr(unsafe.Pointer(addr)), uintptr(sockLen))
		return C.setErrno(C.int(ret))
	}
	err := syscall.SetNonblock(fd, false)
	if err != nil {
		log.Printf("[fd:%v] syscall.SetNonblock failed: %v", fd, err)
		return C.setErrno(errno(err))
	}
	var errCh = make(chan error, 1)
	var proxyUsed *ProxyAddr
	conn := NewFdConn(fd)
	defer conn.Close()
	if config.GetProxyCount() == 0 || config.ShouldNotProxy(dialAddr.IP) {
		go func() {
			err := syscall.Connect(fd, sockAddr)
			if err == nil {
				log.Printf("[fd:%v] direct connect success: %v -> %v", fd, conn.LocalAddr(), dialAddr)
			}
			errCh <- err
		}()
	} else {
		proxyUsed = config.GetProxyAddr()
		go func() {
			err = syscall.Connect(fd, proxyUsed.Sockaddr())
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
			_, err = dialer.Dial("tcp", dialAddr.String())
			if err != nil {
				log.Printf("[fd:%v] dialer Dial %v failed: %v", fd, dialAddr, err)
				errCh <- err
				return
			}
			log.Printf("[fd:%v] proxy connect success: %v -> %v -> %v", fd, conn.LocalAddr(), proxyUsed, dialAddr)
			errCh <- nil
		}()
	}
	timer := time.NewTimer(config.GetConnectTimeouts())
	select {
	case <-timer.C:
		err = syscall.ETIMEDOUT
		log.Printf("[fd:%v] timeout: %v", fd, err)
	case err = <-errCh:
		timer.Stop()
	}
	if err != nil {
		if proxyUsed == nil {
			log.Printf("[fd:%v] direct connect to %v failed %v", fd, dialAddr, err)
		} else {
			log.Printf("[fd:%v] connect to %v using proxy %v failed: %v", fd, dialAddr, proxyUsed, err)
		}
		return C.setErrno(errno(err))
	}
	return 0
}

//export close
func close(fdc C.int) C.int {
	fd := int(fdc)
	conn := NewFdConn(fd)
	remoteAddr := conn.RemoteAddr() // BUG: remoteAddr may be nil even if fd is not closed on Linux
	if opt, _ := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TYPE); opt == syscall.SOCK_STREAM {
		log.Printf("[fd:%v] close conn %v -> %v", fd, conn.LocalAddr(), remoteAddr)
	}
	return C.setErrno(errno(syscall.Close(fd)))
}

//export accept
func accept(fdc C.int, addr *C.struct_sockaddr, sockLen *C.socklen_t) C.int {
	newFD, _, errno := syscall.Syscall(syscall.SYS_ACCEPT, uintptr(fdc),
		uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(sockLen)))
	if errno != 0 {
		return C.setErrno(C.int(errno))
	}
	conn := NewFdConn(int(newFD))
	log.Printf("[fd:%v] accept conn %v -> %v", newFD, conn.LocalAddr(), conn.RemoteAddr())
	return C.int(newFD)
}
