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

// Config holds httpit settings
type Config struct {
	// Connections indicates how many tcp connections are used concurrently
	Connections int
	// Count is numbers of request in one benchmark round
	Count int
	// Qps specifies the highest value for a fixed benchmark, but the real qps
	// may lower than it
	Qps int
	// Duration means benchmark duration, it's ignored if Count is specified
	Duration time.Duration
	// Timeout indicates socket/request timeout
	Timeout time.Duration
	// Url is the benchmark target
	Url string
	// Method is the http method
	Method string
	// Args can set data handily for form and json request
	Args []string
	// Headers indicates http headers
	Headers []string
	// Host can override Host header in request
	Host string
	// DisableKeepAlives sets Connection header to 'close'
	DisableKeepAlives bool
	// Body is request body
	Body string
	// File indicates that read request body from a file
	File string
	// Stream indicates using stream body
	Stream bool
	// JSON indicates send a JSON request
	JSON bool
	// JSON indicates send a Form request
	Form bool
	// Insecure skips tls verification
	Insecure bool
	// Cert indicates path to the client's TLS Certificate
	Cert string
	// Cert indicates path to the client's TLS Certificate private key
	Key string
	// HttpProxy indicates an http proxy address
	HttpProxy string
	// SocksProxy indicates an socks proxy address
	SocksProxy string
	// Pipeline if true, will use fasthttp PipelineClient
	Pipeline bool
	// Follow if true, follow 30x location redirects in debug mode
	Follow bool
	// MaxRedirects indicates maximum redirect count of following 30x,
	// default is 30 (only works if Follow is true)
	MaxRedirects int
	// Debug if true, only send request once and show request and response detail
	Debug bool

	throughput int64
	body       []byte
	isTLS      bool
	addr       string
	tlsConf    *tls.Config
}

func (c *Config) doer() clientDoer {
	if c.Pipeline {
		return &fasthttp.PipelineClient{
			Name:        "httpit/" + Version,
			Addr:        c.addr,
			Dial:        c.getDialer(),
			IsTLS:       c.isTLS,
			TLSConfig:   c.tlsConf,
			MaxConns:    c.Connections,
			ReadTimeout: c.Timeout,
			Logger:      discardLogger{},
		}
	}
	return c.hostClient()
}

func (c *Config) hostClient() *fasthttp.HostClient {
	return &fasthttp.HostClient{
		Name:        "httpit/" + Version,
		Addr:        c.addr,
		Dial:        c.getDialer(),
		IsTLS:       c.isTLS,
		TLSConfig:   c.tlsConf,
		MaxConns:    c.Connections,
		ReadTimeout: c.Timeout,
	}
}

func (c *Config) setReqBasic(req *fasthttp.Request) (err error) {
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
	if c.Body != "" {
		c.body = []byte(c.Body)
	}

	if c.File != "" {
		c.body, err = ioutil.ReadFile(filepath.Clean(c.File))
	}

	if !c.Stream {
		// set constant body
		req.SetBody(c.body)
	}

	return
}

// parseArgs gets body from extra args
func (c *Config) parseArgs() {
	if len(c.Args) == 0 {
		return
	}

	isJson := true
	for _, arg := range c.Args {
		formEqualIndex := strings.Index(arg, "=")
		jsonEqualIndex := strings.Index(arg, ":=")
		// no "=" or "=" is before ":="
		if formEqualIndex == -1 || jsonEqualIndex == -1 || formEqualIndex < jsonEqualIndex {
			isJson = false
		}
	}

	if isJson {
		c.JSON = true
		c.body = append(c.body, '{')
		for ii, arg := range c.Args {
			i := strings.Index(arg, ":=")
			k, v := strings.TrimSpace(arg[:i]), strings.TrimSpace(arg[i+2:])
			c.body = append(c.body, '"')
			c.body = append(c.body, k...)
			c.body = append(c.body, '"', ':')
			b := needQuote(v)
			if b {
				c.body = append(c.body, '"')
			}
			c.body = append(c.body, v...)
			if b {
				c.body = append(c.body, '"')
			}
			if ii < len(c.Args)-1 {
				c.body = append(c.body, ',')
			}
		}
		c.body = append(c.body, '}')
	} else {
		c.Form = true
		c.Method = fasthttp.MethodPost
		formArgs := fasthttp.AcquireArgs()
		for _, arg := range c.Args {
			i := strings.Index(arg, "=")
			if i == -1 {
				formArgs.AddNoValue(strings.TrimSpace(arg))
			} else {
				formArgs.Add(strings.TrimSpace(arg[:i]), strings.TrimSpace(arg[i+1:]))
			}
		}
		c.body = formArgs.AppendBytes(c.body)
		fasthttp.ReleaseArgs(formArgs)
	}
}

func needQuote(v string) bool {
	if vv := strings.ToLower(v); vv == "false" || vv == "true" {
		return false
	}
	if _, err := strconv.Atoi(v); err == nil {
		return false
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return false
	}

	l := len(v)
	if l <= 1 {
		return true
	}

	if (v[0] == '[' && v[l-1] == ']') || (v[0] == '{' && v[l-1] == '}') {
		return false
	}

	return true
}

func (c *Config) setReqHeader(req *fasthttp.Request) (err error) {
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

	return
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
