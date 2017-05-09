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
	"sync"
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
	config.SetProxyAddrs(strings.Split(envProxy[strings.Index(envProxy, "=")+1:], ","))
	config.SetNoProxies(strings.Split(envNotProxies[strings.Index(envNotProxies, "=")+1:], ","))
	config.SetConnectTimeouts(envConnectTimeouts[strings.Index(envConnectTimeouts, "=")+1:])
}

func main() {
}

var config = &Config{}

type Config struct {
	lock            sync.RWMutex
	notProxies      []*net.IPNet
	connectTimeouts time.Duration
	proxyAddrs      []ProxyAddr
}

type ProxyAddr struct {
	proxy.Auth
	AddrStr      string
	ResolvedAddr *net.TCPAddr
}

func (p ProxyAddr) Sockaddr() (addr syscall.Sockaddr) {
	naddr, err := net.ResolveTCPAddr("tcp", p.AddrStr)
	if err != nil {
		log.Println("resolve proxy addr failed", p.AddrStr, err, "use saved addr:", p.ResolvedAddr)
		naddr = p.ResolvedAddr
	} else {
		p.ResolvedAddr = naddr
	}
	if ip4 := naddr.IP.To4(); ip4 != nil {
		var proxyIp4 [4]byte
		copy(proxyIp4[:], ip4)
		return &syscall.SockaddrInet4{
			Addr: proxyIp4,
			Port: naddr.Port,
		}
	} else if ip6 := naddr.IP.To16(); ip6 != nil {
		log.Println("not support ipv6 proxy addr", p.AddrStr, p.ResolvedAddr)
	}
	return
}

func (p ProxyAddr) String() string {
	if p.AddrStr == p.ResolvedAddr.String() {
		return p.AddrStr
	}
	return fmt.Sprintf("%v(%v)", p.AddrStr, p.ResolvedAddr.IP)
}

func (p *Config) SetProxyAddrs(addrs []string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, ipAddr := range addrs {
		var proxyAddr ProxyAddr
		u, err := url.Parse("socks5://" + strings.TrimSpace(ipAddr))
		if err != nil {
			log.Println("parse proxy addr failed", ipAddr, err)
			continue
		}
		if u.User != nil {
			proxyAddr.User = u.User.Username()
			proxyAddr.Password, _ = u.User.Password()
		}
		naddr, err := net.ResolveTCPAddr("tcp", u.Host)
		if err != nil || naddr.IP.To4() == nil {
			log.Println("resolve proxy addr failed", ipAddr, err, naddr.IP)
			continue
		}
		proxyAddr.AddrStr = u.Host
		proxyAddr.ResolvedAddr = naddr
		log.Println("add proxy:", ipAddr)
		p.proxyAddrs = append(p.proxyAddrs, proxyAddr)
	}
	if len(p.proxyAddrs) == 0 {
		log.Println("no proxy available")
	}
}

func (p *Config) GetProxyCount() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.proxyAddrs)
}

func (p *Config) GetProxyAddr() ProxyAddr {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.proxyAddrs[rand.Intn(len(p.proxyAddrs))]
}

func (p *Config) SetConnectTimeouts(timeout string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	t, err := strconv.Atoi(strings.TrimSpace(timeout))
	if err != nil {
		t = 3000
	}
	p.connectTimeouts = time.Duration(t) * time.Millisecond
	log.Println("set connect timeout to", p.connectTimeouts)
}
func (p *Config) GetConnectTimeouts() time.Duration {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.connectTimeouts
}

func (p *Config) ShouldNotProxy(ip net.IP) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, ipnet := range p.notProxies {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func (p *Config) SetNoProxies(addrs []string) {
	p.lock.Lock()
	defer p.lock.Unlock()
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
		p.notProxies = append(p.notProxies, ipnet)
	}
}
