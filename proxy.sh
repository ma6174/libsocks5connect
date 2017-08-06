#!/bin/bash

export socks5_proxy=127.0.0.1:7070
export not_proxy=127.0.0.0/8
export proxy_timeout_ms=1000
export proxy_no_log=false
export libsocks5connect=./libsocks5connect.so

[[ x$1 = x"-f" ]] && source $2 && shift 2

case `uname -s` in
	Darwin)
		export DYLD_FORCE_FLAT_NAMESPACE=1 
		export DYLD_INSERT_LIBRARIES=${libsocks5connect}
		;;

	Linux)
		export LD_PRELOAD=${libsocks5connect}
		;;

	*)
esac

exec "$@"
