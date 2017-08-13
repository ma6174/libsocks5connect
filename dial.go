package main

/*
#include <sys/types.h>
#include <sys/socket.h>
int connect_proxy(int fd, const struct sockaddr * addr, socklen_t sockLen);
int connect(int fd, const struct sockaddr *addr, socklen_t len) {
	return connect_proxy(fd, addr, len);
}
*/
import "C"
import (
	"errors"
	"net"
	"sync"
	"syscall"
	"time"
)

var errClosed = errors.New("connection closed")

var _ net.Conn = NewFdConn(0)

func NewFdConn(fd int) *fdConn {
	return &fdConn{fd: fd}
}

type fdConn struct {
	fd       int
	isClosed bool
	lock     sync.RWMutex
}

func (s *fdConn) Dial(network, addr string) (c net.Conn, err error) {
	return s, nil
}

func (s *fdConn) Write(p []byte) (n int, err error) {
	s.lock.RLock()
	if s.isClosed {
		s.lock.RUnlock()
		return 0, errClosed
	}
	s.lock.RUnlock()
	return syscall.Write(s.fd, p)
}

func (s *fdConn) Read(p []byte) (n int, err error) {
	s.lock.RLock()
	if s.isClosed {
		s.lock.RUnlock()
		return 0, errClosed
	}
	s.lock.RUnlock()
	return syscall.Read(s.fd, p)
}

func (s *fdConn) Close() (err error) {
	s.lock.Lock()
	s.isClosed = true
	s.lock.Unlock()
	return
}
func (s *fdConn) LocalAddr() net.Addr {
	sa, _ := syscall.Getsockname(s.fd)
	return netAddr(sa)
}

func netAddr(sa syscall.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *syscall.SockaddrInet4:
		return &net.TCPAddr{IP: sa.Addr[0:], Port: sa.Port}
	case *syscall.SockaddrInet6:
		return &net.TCPAddr{
			IP:   sa.Addr[0:],
			Port: sa.Port,
			// Zone: zoneToString(int(sa.ZoneId)),
		}
	}
	return nil
}

func (s *fdConn) RemoteAddr() net.Addr {
	sa, _ := syscall.Getpeername(s.fd)
	return netAddr(sa)
}
func (s *fdConn) SetDeadline(t time.Time) error {
	return nil
}
func (s *fdConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (s *fdConn) SetWriteDeadline(t time.Time) error {
	return nil
}
