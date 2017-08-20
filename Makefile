GIT_VERSION := $(shell git rev-parse HEAD)
all:
	go build -i -ldflags "-X 'main.version=git:$(GIT_VERSION)'" -buildmode=c-shared -o libsocks5connect.so

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
