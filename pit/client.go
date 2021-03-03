package pit

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type client interface {
	do(int) (int, time.Duration, error)
}

type clientDoer interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type clientConfig struct {
	method            string
	url               string
	headers           headers
	host              string
	stream            bool
	body              []byte
	http2             bool
	maxConns          int
	timeout           time.Duration
	tlsConfig         *tls.Config
	disableKeepAlives bool
	throughput        *int64
}

type fasthttpClient struct {
	doer    clientDoer
	reqs    []*fasthttp.Request
	resps   []*fasthttp.Response
	readers []*bytes.Reader
	body    []byte
}

func newFasthttpClient(cc clientConfig) (client, error) {
	c := &fasthttpClient{
		reqs:    make([]*fasthttp.Request, cc.maxConns, cc.maxConns),
		resps:   make([]*fasthttp.Response, cc.maxConns, cc.maxConns),
		readers: make([]*bytes.Reader, cc.maxConns, cc.maxConns),
		body:    cc.body,
	}

	isTLS, addr, err := getIsTLSAndAddr(cc.url)
	if err != nil {
		return nil, err
	}

	for i := 0; i < cc.maxConns; i++ {
		req := fasthttp.AcquireRequest()
		req.Header.SetMethod(cc.method)
		req.SetRequestURI(cc.url)
		if err = cc.headers.writeToFasthttp(req); err != nil {
			return nil, err
		}
		if cc.stream {
			c.readers[i] = bytes.NewReader(nil)
		} else {
			// set constant body
			req.SetBody(cc.body)
		}
		if cc.disableKeepAlives {
			req.Header.ConnectionClose()
		}
		if cc.host != "" {
			req.URI().SetHost(cc.host)
		}
		c.reqs[i] = req
		c.resps[i] = fasthttp.AcquireResponse()
	}

	c.doer = &fasthttp.HostClient{
		Addr:                          addr,
		Dial:                          fasthttpDialFunc(cc.throughput, cc.timeout),
		IsTLS:                         isTLS,
		TLSConfig:                     cc.tlsConfig,
		MaxConns:                      cc.maxConns,
		ReadTimeout:                   cc.timeout,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
	}

	return c, nil
}

func (c *fasthttpClient) do(i int) (code int, latency time.Duration, err error) {
	var (
		req    = c.reqs[i]
		resp   = c.resps[i]
		reader = c.readers[i]
	)

	if reader != nil {
		reader.Reset(c.body)
		req.SetBodyStream(reader, -1)
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

func getIsTLSAndAddr(url string) (isTLS bool, addr string, err error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
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
