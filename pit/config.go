package pit

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type Config struct {
	Connections        int
	Count              int
	Duration           time.Duration
	Timeout            time.Duration
	Url                string
	Method             string
	Args               []string
	Headers            []string
	Host               string
	DisableKeepAlives  bool
	Body               string
	File               string
	Stream             bool
	JSON               bool
	Form               bool
	MultipartForm      bool
	MultipartFormFiles []string
	Boundary           string
	Insecure           bool
	Cert               string
	Key                string
	HttpProxy          string
	SocksProxy         string
	Pipeline           bool
	Follow             bool
	MaxRedirects       int
	Debug              bool

	throughput int64
	body       []byte
	isTLS      bool
	addr       string
	tlsConf    *tls.Config
}

func (c *Config) doer() clientDoer {
	if c.Pipeline {
		return &fasthttp.PipelineClient{
			Addr:        c.addr,
			Dial:        c.getDialer(),
			IsTLS:       c.isTLS,
			TLSConfig:   c.tlsConf,
			MaxConns:    c.Connections,
			ReadTimeout: c.Timeout,
			//DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing: true,
			Logger:                 discardLogger{},
		}
	}
	return c.hostClient()
}

func (c *Config) hostClient() *fasthttp.HostClient {
	return &fasthttp.HostClient{
		Addr:                          c.addr,
		Dial:                          c.getDialer(),
		IsTLS:                         c.isTLS,
		TLSConfig:                     c.tlsConf,
		MaxConns:                      c.Connections,
		ReadTimeout:                   c.Timeout,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
	}
}

func (c *Config) setReqBasic(req *fasthttp.Request) (err error) {
	req.Header.DisableNormalizing()
	req.Header.SetMethod(c.Method)
	req.SetRequestURI(c.Url)

	uri := req.URI()
	host := uri.Host()

	scheme := uri.Scheme()
	if bytes.Equal(scheme, strHTTPS) {
		c.isTLS = true
	} else if !bytes.Equal(scheme, strHTTP) {
		err = fmt.Errorf("unsupported protocol %q. http and https are supported", scheme)
		return
	}

	c.addr = addMissingPort(string(host), c.isTLS)

	return
}

var (
	strHTTP  = []byte("http")
	strHTTPS = []byte("https")
)

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

func (c *Config) setReqBody(req *fasthttp.Request) (err error) {
	if c.File == "" {
		c.body = []byte(c.Body)
	} else {
		c.body, err = ioutil.ReadFile(filepath.Clean(c.File))
	}

	// TODO parse files and args

	if !c.Stream {
		// set constant body
		req.SetBody(c.body)
	}

	return
}

func (c *Config) setReqHeader(req *fasthttp.Request) (err error) {
	// TODO parse args

	if err = headers(c.Headers).writeToFasthttp(req); err != nil {
		return
	}

	if c.DisableKeepAlives {
		req.Header.SetConnectionClose()
	}
	if c.Host != "" {
		req.URI().SetHost(c.Host)
	}

	if c.JSON {
		req.Header.SetContentType(MIMEApplicationJSON)
	}
	if c.Form {
		req.Header.SetContentType(MIMEApplicationForm)
	}
	if c.MultipartForm {
		req.Header.SetMultipartFormBoundary(c.Boundary)
	}

	return nil
}

func (c *Config) getDialer() fasthttp.DialFunc {
	if c.HttpProxy != "" {
		return fasthttpHttpProxyDialer(&c.throughput, c.HttpProxy, c.Timeout)
	}
	if c.SocksProxy != "" {
		return fasthttpSocksProxyDialer(&c.throughput, c.HttpProxy)
	}

	return fasthttpDialer(&c.throughput, c.Timeout)
}

/* #nosec G402 */
func (c *Config) getTlsConfig() (conf *tls.Config, err error) {
	var certs []tls.Certificate
	if certs, err = readClientCert(c.Cert, c.Key); err != nil {
		return
	}
	conf = &tls.Config{
		Certificates:       certs,
		InsecureSkipVerify: c.Insecure,
	}

	return
}

func readClientCert(certPath, keyPath string) (certs []tls.Certificate, err error) {
	if certPath == "" && keyPath == "" {
		return
	}

	var cert tls.Certificate
	if cert, err = tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		return
	}

	certs = append(certs, cert)

	return
}

func (c *Config) getMaxRedirects() int {
	if !c.Follow {
		return 0
	}

	n := c.MaxRedirects
	if n <= 0 {
		n = defaultMaxRedirects
	}

	return n
}
