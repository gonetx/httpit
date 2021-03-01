package pit

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
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

var fasthttpDialFunc = func(throughput *int64, timeout time.Duration) func(string) (net.Conn, error) {
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

var httpDialContextFunc = func(throughput *int64) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
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
