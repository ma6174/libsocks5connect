package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
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

var version, configAddr string

func init() {
	rand.Seed(time.Now().UnixNano())

	envNoLog := os.Getenv("proxy_no_log") // false
	config.SetNoLog(envNoLog[strings.Index(envNoLog, "=")+1:])
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
	log.Println("libsocks5connect loaded, version:", version)

	envProxy := os.Getenv("socks5_proxy") // user:pass@192.168.1.1:1080,user:pass@192.168.1.2:1080
	config.SetProxyAddrs(strings.Split(envProxy[strings.Index(envProxy, "=")+1:], ","))

	envNotProxies := os.Getenv("not_proxy") // 127.0.0.0/8,192.168.1.0/24
	config.SetNoProxies(strings.Split(envNotProxies[strings.Index(envNotProxies, "=")+1:], ","))

	envConnectTimeouts := os.Getenv("proxy_timeout_ms") // 1000
	config.SetConnectTimeouts(envConnectTimeouts[strings.Index(envConnectTimeouts, "=")+1:])

	envNoConfigServer := os.Getenv("no_config_server") // false
	envNoConfigServer = strings.ToLower(strings.TrimSpace(
		envNoConfigServer[strings.Index(envNoConfigServer, "=")+1:]))
	if envNoConfigServer != "true" && envNoConfigServer != "1" {
		go config.Listen()
	}
}

func main() {
}

var config = &Config{}

type Config struct {
	lock            sync.RWMutex
	notProxies      []*net.IPNet
	connectTimeouts time.Duration
	proxyAddrs      []ProxyAddr
	proxyNoLog      bool
}

func (p *Config) String() string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return fmt.Sprintf("%v: %v\n%v=%v\n%v=%v\n%v=%v\n%v=%v\n",
		"version", version,
		"proxy_no_log", p.IsProxyNoLog(),
		"socks5_proxy", strings.Join(p.GetProxyAddrs(), ","),
		"not_proxy", strings.Join(p.GetNoProxies(), ","),
		"proxy_timeout_ms", uint64(p.GetConnectTimeouts()/time.Millisecond),
	)
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
	return fmt.Sprintf("%v(%v)", p.AddrStr, p.ResolvedAddr)
}

func (p *Config) SetProxyAddrs(addrs []string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, ipAddr := range addrs {
		ipAddr = strings.TrimSpace(ipAddr)
		if len(ipAddr) == 0 {
			continue
		}
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

func (p *Config) GetProxyAddr() *ProxyAddr {
	p.lock.RLock()
	defer p.lock.RUnlock()
	tmpAddr := p.proxyAddrs[rand.Intn(len(p.proxyAddrs))]
	return &tmpAddr
}
func (p *Config) GetProxyAddrs() (addrs []string) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, addr := range p.proxyAddrs {
		addrs = append(addrs, addr.AddrStr)
	}
	return
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

func (p *Config) SetNoLog(isNoLog string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	isNoLog = strings.ToLower(strings.TrimSpace(isNoLog))
	if isNoLog == "true" || isNoLog == "1" {
		log.SetOutput(ioutil.Discard)
		p.proxyNoLog = true
	} else {
		log.SetOutput(os.Stderr)
		p.proxyNoLog = false
	}
}
func (p *Config) IsProxyNoLog() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.proxyNoLog
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

func (p *Config) GetNoProxies() (addrs []string) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, addr := range p.notProxies {
		addrs = append(addrs, addr.String())
	}
	return
}

func (p *Config) Listen() {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Println("listen failed", err)
		return
	}
	configAddr = ln.Addr().String()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept failed", err)
			continue
		}
		go p.UpdateConfigFromConn(conn)
	}
}

func (p *Config) UpdateConfigFromConn(conn net.Conn) {
	defer conn.Close()
	log.Println("config server new connection from:", conn.RemoteAddr())
	fmt.Fprintf(conn, "current config:\n%v\n\n", p)
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		sp := strings.SplitN(scanner.Text(), "=", 2)
		if len(sp) < 2 {
			log.Println("invalid config")
			fmt.Fprintf(conn, "invalid config %#v\n", scanner.Text())
			fmt.Fprintf(conn, "current config:\n%v\n\n", p)
			continue
		}
		switch sp[0] {
		case "socks5_proxy", "export socks5_proxy":
			config.SetProxyAddrs(strings.Split(sp[1], ","))
			fmt.Fprintf(conn, "OK, current proxyaddrs: %v\n",
				strings.Join(config.GetProxyAddrs(), ","))
		case "not_proxy", "export not_proxy":
			config.SetNoProxies(strings.Split(sp[1], ","))
			fmt.Fprintf(conn, "OK, current no proxy addrs: %v\n",
				strings.Join(config.GetNoProxies(), ","))
		case "proxy_timeout_ms", "export proxy_timeout_ms":
			config.SetConnectTimeouts(sp[1])
			fmt.Fprintf(conn, "OK, current proxy timeouts: %v\n",
				uint64(p.GetConnectTimeouts()/time.Millisecond))
		case "proxy_no_log", "export proxy_no_log":
			config.SetNoLog(sp[1])
			fmt.Fprintf(conn, "OK, proxy_no_log: %v\n",
				config.IsProxyNoLog())
		default:
			log.Println("unknown config")
			fmt.Fprintf(conn, "unknown config %#v\n", scanner.Text())
			fmt.Fprintf(conn, "current config:\n%v\n\n", p)
		}
	}
}
