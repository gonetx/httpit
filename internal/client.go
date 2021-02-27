package internal

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/http2"

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
	headers           Headers
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
		cc.headers.WriteToFasthttp(req)
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

	if cc.disableKeepAlives {
		c.doer = &fasthttp.HostClient{
			Addr:                          addr,
			NoDefaultUserAgentHeader:      false,
			Dial:                          fasthttpDialFunc(cc.throughput, cc.timeout),
			DialDualStack:                 false,
			IsTLS:                         isTLS,
			TLSConfig:                     cc.tlsConfig,
			MaxConns:                      cc.maxConns,
			ReadTimeout:                   cc.timeout,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		}
	} else {
		c.doer = &fasthttp.PipelineClient{
			Addr:        addr,
			MaxConns:    cc.maxConns,
			Dial:        fasthttpDialFunc(cc.throughput, cc.timeout),
			IsTLS:       isTLS,
			TLSConfig:   cc.tlsConfig,
			ReadTimeout: cc.timeout,
		}
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

type httpClient struct {
	client  *http.Client
	reqs    []*http.Request
	readers []*bytes.Reader
	body    []byte
}

func newHttpClient(cc clientConfig) (client, error) {
	c := &httpClient{
		reqs:    make([]*http.Request, 0, cc.maxConns),
		readers: make([]*bytes.Reader, 0, cc.maxConns),
	}

	var err error

	for i := 0; i < cc.maxConns; i++ {
		if c.reqs[i], err = http.NewRequest(cc.method, cc.url, nil); err != nil {
			return nil, fmt.Errorf("failed to new request: %w", err)
		}
		req := c.reqs[i]

		cc.headers.WriteToHttp(req)

		if cc.host != "" {
			req.Host = cc.host
		}
	}

	transport := &http.Transport{
		TLSClientConfig:     cc.tlsConfig,
		MaxIdleConnsPerHost: cc.maxConns,
		DisableKeepAlives:   cc.disableKeepAlives,
	}
	transport.DialContext = httpDialContextFunc(cc.throughput)
	if cc.http2 {
		if err = http2.ConfigureTransport(transport); err != nil {
			return nil, fmt.Errorf("failed to setup http2: %w", err)
		}
	} else {
		transport.TLSNextProto = make(
			map[string]func(authority string, c *tls.Conn) http.RoundTripper,
		)
	}

	c.client = &http.Client{
		Transport: transport,
		Timeout:   cc.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return c, nil
}

func (c *httpClient) do(i int) (code int, latency time.Duration, err error) {
	req := c.reqs[i]

	reader := c.readers[i]
	reader.Reset(c.body)
	req.Body = ioutil.NopCloser(reader)

	var resp *http.Response
	start := time.Now()

	if resp, err = c.client.Do(req); err != nil {
		return
	}

	if err = resp.Body.Close(); err != nil {
		return
	}

	code = resp.StatusCode
	latency = time.Since(start)

	return
}
