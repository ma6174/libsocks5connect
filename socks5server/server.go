package main

import (
	"log"

	socks5 "github.com/armon/go-socks5"
)

func main() {
	svr, err := socks5.New(&socks5.Config{})
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(svr.ListenAndServe("tcp", "127.0.0.1:1080"))
}
