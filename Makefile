all:
	go build -i -buildmode=c-shared -o libsocks5connect.so

clean:
	rm libsocks5connect.so
	rm libsocks5connect.h
