GIT_VERSION := $(shell git rev-parse HEAD)
all:
	go build -i -ldflags "-w -s -X 'main.version=git:$(GIT_VERSION)'" -buildmode=c-shared -o libsocks5connect.so
	cd ./cmd/socks5proxy && go generate && go build

clean:
	rm libsocks5connect.so
	rm libsocks5connect.h

update-dep:
	go get -v github.com/Masterminds/glide
	go get github.com/sgotti/glide-vc
	glide update
	glide vc

runSocks5Server:
	go build socks5server/server.go
	./server &

test: all runSocks5Server
	go test -v -tags test -cover
	./proxy.sh -f proxy_test.conf python2 -c 'import urllib2;print(len(urllib2.urlopen("http://golang.org").read()))'
	./proxy.sh -f proxy_test.conf curl -I -L http://golang.org
	export socks5_proxy=127.0.0.1:1080
	export not_proxy=
	export proxy_timeout_ms=1000
	export proxy_no_log=false
	./cmd/socks5proxy/socks5proxy python2 -c 'import urllib2;print(len(urllib2.urlopen("http://golang.org").read()))'
	./cmd/socks5proxy/socks5proxy curl -I -L http://golang.org
