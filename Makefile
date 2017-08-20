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
	nohup ./server &

test: all runSocks5Server
	sleep 0.1
	./proxy.sh -f proxy_test.conf python2 -c 'import urllib2;print(len(urllib2.urlopen("http://golang.org").read()))'
	cat nohup.out
	go test -v -tags test -cover
