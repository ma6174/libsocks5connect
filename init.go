package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/proxy"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
	rand.Seed(time.Now().UnixNano())
	log.Println("proxy init", os.Args)
	envProxy := os.Getenv("socks5_proxy")               // user:pass@192.168.1.1:1080,user:pass@192.168.1.2:1080
	envNotProxies := os.Getenv("not_proxy")             // 127.0.0.0/8,192.168.1.0/24
	envConnectTimeouts := os.Getenv("proxy_timeout_ms") // 1000
	initPorxyAddrs(strings.Split(envProxy[strings.Index(envProxy, "=")+1:], ","))
	initNotProxies(strings.Split(envNotProxies[strings.Index(envNotProxies, "=")+1:], ","))
	initConnectTimeouts(envConnectTimeouts[strings.Index(envConnectTimeouts, "=")+1:])
}

func main() {
}

type proxyAddr struct {
	AddrStr  string
	Auth     proxy.Auth
	SockAddr syscall.Sockaddr
}

var connectTimeouts time.Duration

func initConnectTimeouts(timeout string) {
	t, err := strconv.Atoi(strings.TrimSpace(timeout))
	if err != nil {
		t = 3000
	}
	connectTimeouts = time.Duration(t) * time.Millisecond
	log.Println("set connect timeout", connectTimeouts)
}

var proxyAddrs []*proxyAddr

func initPorxyAddrs(proxies []string) {
	for _, ipAddr := range proxies {
		u, err := url.Parse("socks5://" + strings.TrimSpace(ipAddr))
		if err != nil {
			log.Println("parse proxy addr failed", ipAddr, err)
			continue
		}
		var auth proxy.Auth
		if u.User != nil {
			auth.User = u.User.Username()
			auth.Password, _ = u.User.Password()
		}
		naddr, err := net.ResolveTCPAddr("tcp", ipAddr)
		if err != nil {
			log.Println("resolve proxy addr failed", ipAddr, err)
			continue
		}
		if naddr.IP.To4() == nil {
			continue
		}
		if naddr.String() != ipAddr {
			ipAddr = fmt.Sprintf("%s(%s)", ipAddr, naddr.String())
		}
		log.Println("add proxy:", ipAddr)
		var proxyIp4 [4]byte
		copy(proxyIp4[:], naddr.IP.To4())
		proxyAddrs = append(proxyAddrs, &proxyAddr{
			Auth:    auth,
			AddrStr: ipAddr,
			SockAddr: &syscall.SockaddrInet4{
				Addr: proxyIp4,
				Port: naddr.Port,
			},
		})
	}
	if len(proxyAddrs) == 0 {
		log.Println("no proxy available")
	}
}

type NotProxies []*net.IPNet

func (p NotProxies) Contains(ip net.IP) bool {
	for _, ipnet := range p {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

var notProxies NotProxies

func initNotProxies(addrs []string) {
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if len(addr) == 0 {
			continue
		}
		_, ipnet, err := net.ParseCIDR(addr)
		if err != nil {
			log.Println("parse ipnet failed", err, addr)
			continue
		}
		log.Println("add not proxy addr:", addr)
		notProxies = append(notProxies, ipnet)
	}
}
