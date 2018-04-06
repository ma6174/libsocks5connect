package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

//go:generate bash gen.sh

func libExt() string {
	switch runtime.GOOS {
	case "darwin":
		return ".dylib"
	case "linux":
		return ".so"
	}
	return ""
}

var libFile = filepath.Join("/tmp", "libsocks5connect."+libmd5+libExt())

func writeLib() {
	_, err := os.Stat(libFile)
	if !os.IsNotExist(err) {
		return
	}
	f, err := os.OpenFile(libFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return
	}
	defer f.Close()
	r, err := gzip.NewReader(bytes.NewReader(lib))
	if err != nil {
		return
	}
	io.Copy(f, r)
}

func main() {
	writeLib()
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	switch runtime.GOOS {
	case "darwin":
		cmd.Env = append(cmd.Env, "DYLD_FORCE_FLAT_NAMESPACE=1")
		cmd.Env = append(cmd.Env, "DYLD_INSERT_LIBRARIES="+libFile)
	case "linux":
		cmd.Env = append(cmd.Env, "LD_PRELOAD="+libFile)
	}
	err := cmd.Run()
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			os.Exit(int(errno))
		} else {
			os.Exit(-1)
		}
	}
}
