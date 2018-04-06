#!/bin/bash

echo -e 'package main\n' > bin.go
echo -e 'var libmd5 = "'`md5sum ../../libsocks5connect.so | awk '{print $1}'`'"' >> bin.go
echo -e 'var lib = []byte{' >> bin.go
cat ../../libsocks5connect.so | gzip | xxd -i | sed "s/[^,]$/}/g" >> bin.go
