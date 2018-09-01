package main

import (
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"os"
	"socks5"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Printf("usage: %s bindaddr onionsocksaddr [i2psocksaddr]\n", os.Args[0])
		return
	}
	var onion, i2p proxy.Dialer
	var err error
	onion, err = proxy.SOCKS5("tcp", os.Args[2], nil, nil)
	if err != nil {
		fmt.Printf("failed to create onion proxy to %s, %s\n", os.Args[2], err.Error())
		return
	}
	if len(args) > 2 {
		i2p, err = proxy.SOCKS5("tcp", os.Args[3], nil, nil)
		if err != nil {
			fmt.Printf("failed to create i2p proxy to %s, %s\n", os.Args[3], err.Error())
			return
		}
	}
	serv, err := socks5.New(&socks5.Config{
		Dial: func(addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if strings.HasSuffix(host, ".i2p") {
				if i2p == nil {
					return onion.Dial("tcp", addr)
				}
				return i2p.Dial("tcp", addr)
			}
			if strings.HasSuffix(host, ".onion") {
				return onion.Dial("tcp", addr)
			}
			return net.Dial("tcp", addr)
		},
	})

	if err != nil {
		fmt.Printf("failed to create socks proxy %s\n", err.Error())
		return
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		fmt.Printf("failed to listen on %s, %s\n", os.Args[1], err.Error())
		return
	}
	fmt.Printf("proxy serving on %s\n", os.Args[1])
	serv.Serve(l)
}
