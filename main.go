package main

import (
	"crypto/tls"
	"fmt"
	"github.com/majestrate/fedproxy/internal/socks5"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

type httpProxyHandler struct {
	onion proxy.Dialer
	i2p   proxy.Dialer
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (h *httpProxyHandler) dialOut(addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(host, ".loki") {
		return net.Dial("tcp", addr)
	}
	if strings.HasSuffix(host, ".i2p") {
		return h.i2p.Dial("tcp", addr)
	}
	return h.onion.Dial("tcp", addr)
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
		w.Header().Del("Transfer-Encoding")
		w.WriteHeader(http.StatusOK)
		conn, _, err := hijacker.Hijack()
		if err != nil {
			outConn.Close()
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		go transfer(conn, outConn)
		go transfer(outConn, conn)
	} else {
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) < 4 {
		fmt.Printf("usage: %s proto bindaddr onionsocksaddr i2psocksaddr\n", os.Args[0])
		return
	}
	usehttp := args[0] == "http"
	onionsock, err := proxy.SOCKS5("tcp", args[2], nil, nil)
	if err != nil {
		fmt.Printf("failed to create upstream proxy to %s, %s", args[2], err.Error())
		return
	}
	i2psock, err := proxy.SOCKS5("tcp", args[3], nil, nil)
	if usehttp {
		serv := &http.Server{
			Addr: args[1],
			Handler: &httpProxyHandler{
				onion: onionsock,
				i2p:   i2psock,
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		fmt.Printf("setting up http proxy at %s\n", serv.Addr)
		err = serv.ListenAndServe()
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	} else {
		serv, err := socks5.New(&socks5.Config{
			Dial: func(addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				host = strings.TrimSuffix(host, ".")
				fmt.Printf("%s\n", host)
				if err != nil {
					return nil, err
				}
				if strings.HasSuffix(host, ".loki") {
					return net.Dial("tcp", addr)
				}
				if strings.HasSuffix(host, ".i2p") {
					return i2psock.Dial("tcp", addr)
				}
				return onionsock.Dial("tcp", addr)
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
		}
		serv.Serve(l)
	}
}
