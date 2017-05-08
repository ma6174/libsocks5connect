package main

/*
#include <netinet/in.h>
int connect_proxy(int fd, const struct sockaddr * addr, socklen_t sockLen);
int connect(int fd, const struct sockaddr *addr, socklen_t len) {
	return connect_proxy(fd, addr, len);
}
*/
import "C"
import (
	"net"
	"syscall"
	"time"
)

var _ net.Conn = fdConn(0)

type fdConn int

func (s fdConn) Dial(network, addr string) (c net.Conn, err error) {
	return s, nil
}

func (s fdConn) Write(p []byte) (n int, err error) {
	return syscall.Write(int(s), p)
}

func (s fdConn) Read(p []byte) (n int, err error) {
	return syscall.Read(int(s), p)
}

func (s fdConn) Close() (err error) {
	return
}
func (s fdConn) LocalAddr() net.Addr {
	return nil
}
func (s fdConn) RemoteAddr() net.Addr {
	return nil
}
func (s fdConn) SetDeadline(t time.Time) error {
	return nil
}
func (s fdConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (s fdConn) SetWriteDeadline(t time.Time) error {
	return nil
}
