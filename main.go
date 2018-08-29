package main

import (
	"socks5"
	"golang.org/x/net/context"
	"golang.org/x/net/proxy"
	"net"
	"os"
	"fmt"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Printf("usage: %s bindaddr upstreamaddr\n", os.Args[0])
		return
	}
	
	upstream, err:= proxy.SOCKS5("tcp", os.Args[2], nil, nil)
	if err != nil {
		fmt.Printf("failed to create upstream proxy to %s, %s", os.Args[2], err.Error())
		return		
	}
	serv, err := socks5.New(&socks5.Config{
		Dial: func(_ context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if strings.HasSuffix(host, ".onion") {
				return upstream.Dial(network, addr)
			}
			return net.Dial(network, addr)
		},
	})

	if err != nil {
		fmt.Printf("failed to create socks proxy %s",  err.Error())
		return		
	}
	
	
	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		fmt.Printf("failed to listen on %s, %s", os.Args[1], err.Error())
		return
	}
	serv.Serve(l)
}
