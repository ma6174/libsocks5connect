package main

//#include <arpa/inet.h>
//#include <errno.h>
//static inline int setErrno(int err) {
//     errno = err;
//     return -1;
//}
import "C"
import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"syscall"
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
	}
	dialAddr := net.IP(ip).String() + ":" + fmt.Sprint(port)
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
	if opt == syscall.SOCK_DGRAM || len(proxyAddrs) == 0 || notProxies.Contains(net.IP(ip)) {
		err = syscall.Connect(int(fd), sockAddr)
		if err != nil {
			log.Printf("direct connect to %v failed %v", dialAddr, err)
			return C.setErrno(errno(err))
		}
		log.Printf("direct connect to %v success", dialAddr)
		return 0
	}
	proxyUsed := proxyAddrs[rand.Intn(len(proxyAddrs))]
	err = syscall.Connect(int(fd), proxyUsed.SockAddr)
	if err != nil {
		log.Printf("connect to %v using proxy %v failed: %v",
			dialAddr, proxyUsed.AddrStr, err)
		return C.setErrno(errno(err))
	}
	dialer, err := proxy.SOCKS5("", "", &proxyUsed.Auth, fdConn(fd))
	if err != nil {
		log.Println("proxy.SOCKS5 failed", err)
		return C.setErrno(errno(err))
	}
	_, err = dialer.Dial("tcp", dialAddr)
	if err != nil {
		log.Printf("dialer Dial %v failed: %v", dialAddr, err)
		return C.setErrno(errno(err))
	}
	log.Printf("connect to %v using proxy %v success", dialAddr, proxyUsed.AddrStr)
	return 0
}
