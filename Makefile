GIT_VERSION := $(shell git rev-parse HEAD)
all:
	go build -i -ldflags "-s -w -X 'main.version=git:$(GIT_VERSION)'" -buildmode=c-shared -o libsocks5connect.so

clean:
	rm libsocks5connect.so
	rm libsocks5connect.h
