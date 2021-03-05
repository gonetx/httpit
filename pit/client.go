package pit

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type client interface {
	do() (int, time.Duration, error)
}

type clientDoer interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type clientConfig struct {
	method            string
	url               string
	headers           headers
	host              string
	body              []byte
	maxConns          int
	timeout           time.Duration
	tlsConfig         *tls.Config
	throughput        *int64
	httpProxy         string
	socksProxy        string
	stream            bool
	http2             bool
	disableKeepAlives bool
	pipeline          bool
}

type fasthttpClient struct {
	doer           clientDoer
	reqPool        sync.Pool
	bodyStreamPool sync.Pool
	rawReq         *fasthttp.Request
	body           []byte
	stream         bool
}

func newFasthttpClient(cc clientConfig) (client, error) {
	c := &fasthttpClient{
		body:           cc.body,
		stream:         cc.stream,
		bodyStreamPool: sync.Pool{New: func() interface{} { return bytes.NewReader(nil) }},
	}

	req := fasthttp.AcquireRequest()
	req.Header.DisableNormalizing()
	req.Header.SetMethod(cc.method)
	req.SetRequestURI(cc.url)
	if err := cc.headers.writeToFasthttp(req); err != nil {
		return nil, err
	}
	if !cc.stream {
		// set constant body
		req.SetBody(cc.body)
	}
	if cc.disableKeepAlives {
		req.Header.ConnectionClose()
	}
	if cc.host != "" {
		req.URI().SetHost(cc.host)
	}

	c.rawReq = req

	c.reqPool = sync.Pool{
		New: func() interface{} {
			req := fasthttp.AcquireRequest()
			c.rawReq.CopyTo(req)
			return req
		},
	}

	isTLS, addr, err := getIsTLSAndAddr(c.rawReq)
	if err != nil {
		return nil, err
	}

	if cc.pipeline {
		c.doer = &fasthttp.PipelineClient{
			Addr:        addr,
			Dial:        getDialer(cc),
			IsTLS:       isTLS,
			TLSConfig:   cc.tlsConfig,
			MaxConns:    cc.maxConns,
			ReadTimeout: cc.timeout,
			//DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing: true,
			Logger:                 discardLogger{},
		}
	} else {
		c.doer = &fasthttp.HostClient{
			Addr:                          addr,
			Dial:                          getDialer(cc),
			IsTLS:                         isTLS,
			TLSConfig:                     cc.tlsConfig,
			MaxConns:                      cc.maxConns,
			ReadTimeout:                   cc.timeout,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		}
	}

	return c, nil
}

func getDialer(cc clientConfig) fasthttp.DialFunc {
	if cc.httpProxy != "" {
		return fasthttpHttpProxyDialer(cc.throughput, cc.httpProxy, cc.timeout)
	}
	if cc.socksProxy != "" {
		return fasthttpSocksProxyDialer(cc.throughput, cc.httpProxy)
	}

	return fasthttpDialer(cc.throughput, cc.timeout)
}

func (c *fasthttpClient) do() (code int, latency time.Duration, err error) {
	var (
		req  = c.reqPool.Get().(*fasthttp.Request)
		resp = fasthttp.AcquireResponse()
	)

	defer func() {
		c.reqPool.Put(req)
		fasthttp.ReleaseResponse(resp)
	}()

	if c.stream {
		bodyStream := c.bodyStreamPool.Get().(*bytes.Reader)
		bodyStream.Reset(c.body)
		req.SetBodyStream(bodyStream, -1)
		c.bodyStreamPool.Put(bodyStream)
	}

	start := time.Now()
	if err = c.doer.Do(req, resp); err != nil {
		return
	}

	code = resp.StatusCode()
	latency = time.Since(start)

	return
}

var (
	strHTTP  = []byte("http")
	strHTTPS = []byte("https")
)

func getIsTLSAndAddr(req *fasthttp.Request) (isTLS bool, addr string, err error) {
	uri := req.URI()
	host := uri.Host()

	scheme := uri.Scheme()
	if bytes.Equal(scheme, strHTTPS) {
		isTLS = true
	} else if !bytes.Equal(scheme, strHTTP) {
		err = fmt.Errorf("unsupported protocol %q. http and https are supported", scheme)
		return
	}

	addr = addMissingPort(string(host), isTLS)

	return
}

func addMissingPort(addr string, isTLS bool) string {
	n := strings.Index(addr, ":")
	if n >= 0 {
		return addr
	}
	port := 80
	if isTLS {
		port = 443
	}
	return net.JoinHostPort(addr, strconv.Itoa(port))
}

type discardLogger struct{}

func (discardLogger) Printf(_ string, _ ...interface{}) {}
