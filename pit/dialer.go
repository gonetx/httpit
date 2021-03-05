package pit

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/proxy"
)

// counterConn counts read bytes and written bytes
type counterConn struct {
	net.Conn
	n *int64
}

func (cc *counterConn) Read(b []byte) (n int, err error) {
	n, err = cc.Conn.Read(b)

	if err == nil {
		atomic.AddInt64(cc.n, int64(n))
	}

	return
}

func (cc *counterConn) Write(b []byte) (n int, err error) {
	n, err = cc.Conn.Write(b)

	if err == nil {
		atomic.AddInt64(cc.n, int64(n))
	}

	return
}

var fasthttpDialer = func(throughput *int64, timeout time.Duration) func(string) (net.Conn, error) {
	dialer := &fasthttp.TCPDialer{}
	return func(address string) (net.Conn, error) {
		conn, err := dialer.DialDualStackTimeout(address, timeout)
		if err != nil {
			return nil, err
		}

		cc := &counterConn{
			Conn: conn,
			n:    throughput,
		}

		return cc, nil
	}
}

func fasthttpHttpProxyDialer(throughput *int64, proxy string, timeout time.Duration) fasthttp.DialFunc {
	var auth string
	if strings.Contains(proxy, "@") {
		split := strings.Split(proxy, "@")
		auth = base64.StdEncoding.EncodeToString([]byte(split[0]))
		proxy = split[1]
	}

	return func(addr string) (net.Conn, error) {
		var conn net.Conn
		var err error
		if timeout == 0 {
			conn, err = fasthttp.Dial(proxy)
		} else {
			conn, err = fasthttp.DialTimeout(proxy, timeout)
		}
		if err != nil {
			return nil, fmt.Errorf("http proxy: %w", err)
		}

		req := "CONNECT " + addr + " HTTP/1.1\r\n"
		if auth != "" {
			req += "Proxy-Authorization: Basic " + auth + "\r\n"
		}
		req += "\r\n"

		if _, err = conn.Write([]byte(req)); err != nil {
			return nil, fmt.Errorf("http proxy: %w", err)
		}

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		res.SkipBody = true

		if err = res.Read(bufio.NewReader(conn)); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("http proxy: %w", err)
		}
		if res.Header.StatusCode() != 200 {
			_ = conn.Close()
			return nil, fmt.Errorf("could not connect to proxy")
		}

		cc := &counterConn{
			Conn: conn,
			n:    throughput,
		}

		return cc, nil
	}
}

func fasthttpSocksProxyDialer(throughput *int64, proxyAddr string) fasthttp.DialFunc {
	var (
		u      *url.URL
		err    error
		dialer proxy.Dialer
	)

	if u, err = url.Parse(proxyAddr); err == nil {
		dialer, err = proxy.FromURL(u, proxy.Direct)
	}

	return func(addr string) (net.Conn, error) {
		if err != nil {
			return nil, fmt.Errorf("socks proxy: %w", err)
		}
		var conn net.Conn
		conn, err = dialer.Dial("tcp", addr)

		return &counterConn{
			Conn: conn,
			n:    throughput,
		}, err
	}
}
