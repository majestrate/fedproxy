package main

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
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

func (h *httpProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		outConn, err := h.dialOut(r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			outConn.Close()
			http.Error(w, "hijack disallowed", http.StatusInternalServerError)
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			outConn.Close()
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		go transfer(conn, outConn)
		go transfer(outConn, conn)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	args := os.Args[1:]
	usehttp := args[0] == "http"
	if len(args) < 2 {
		fmt.Printf("usage: %s proto bindaddr onionsocksaddr\n", os.Args[0])
		return
	}

	upstream, err := proxy.SOCKS5("tcp", args[2], nil, nil)
	if err != nil {
		fmt.Printf("failed to create upstream proxy to %s, %s", args[2], err.Error())
		return
	}
	if usehttp {
		serv := &http.Server{
			Addr: args[1],
			Handler: &httpProxyHandler{
				upstream: upstream,
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		serv.ListenAndServe()
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

		l, err := net.Listen("tcp", args[1])
		if err != nil {
			fmt.Printf("failed to listen on %s, %s", args[1], err.Error())
			return

			serv.Serve(l)
		}
	}
}
