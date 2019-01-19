package main

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"os"
	"socks5"
	"strings"
)

type httpProxyHandler struct {
	upstream proxy.Dialer
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func (h *httpProxyHandler) dialOut(addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(host, ".onion") {
		return h.upstream.Dial("tcp", addr)
	}
	return net.Dial("tcp", addr)
}

func (h *httpProxyHandler) ServeHTTP(w http.ResposneWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		outConn, err := h.dialOut(r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack disallowed", http.StatusInternalServerError)
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	usehttp := os.Args[0] == "http-fedproxy"
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Printf("usage: %s bindaddr upstreamaddr\n", os.Args[0])
		return
	}

	upstream, err := proxy.SOCKS5("tcp", os.Args[2], nil, nil)
	if err != nil {
		fmt.Printf("failed to create upstream proxy to %s, %s", os.Args[2], err.Error())
		return
	}
	if usehttp {
		serv := &http.Server{
			Addr: os.Args[1],
			Handler: &proxyHandler{
				upstream: upstream,
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		http.ListenAndServe(proxyHandler, os.Args[1])
	} else {
		serv, err := socks5.New(&socks5.Config{
			Dial: func(addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				if strings.HasSuffix(host, ".onion") {
					return upstream.Dial("tcp", addr)
				}
				return net.Dial("tcp", addr)
			},
		})

		if err != nil {
			fmt.Printf("failed to create socks proxy %s", err.Error())
			return
		}

		l, err := net.Listen("tcp", os.Args[1])
		if err != nil {
			fmt.Printf("failed to listen on %s, %s", os.Args[1], err.Error())
			return

			serv.Serve(l)
		}
	}
}
