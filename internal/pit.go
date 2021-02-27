package internal

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"sync"
	"time"
)

type Pit struct {
	c Config
	client
	errorMap
	throughput int64
	// HTTP codes
	code1xx uint64
	code2xx uint64
	code3xx uint64
	code4xx uint64
	code5xx uint64
	others  uint64

	wg       sync.WaitGroup
	doneChan chan struct{}

	mut   sync.Mutex
	reqs  int64
	start time.Time

	elapsed time.Duration
}

func NewPit(c Config) *Pit {
	return &Pit{
		c:        c,
		doneChan: make(chan struct{}),
	}
}

func (p *Pit) Run(url string) (err error) {
	if err := p.init(url); err != nil {
		return err
	}

	return errors.New("error")
}

func (p *Pit) init(url string) (err error) {
	cc := clientConfig{
		method:            p.c.Method,
		url:               url,
		headers:           p.c.Headers,
		host:              p.c.Host,
		stream:            p.c.Stream,
		http2:             p.c.Http2,
		maxConns:          p.c.Connections,
		timeout:           p.c.Timeout,
		disableKeepAlives: p.c.DisableKeepAlives,
		throughput:        &p.throughput,
	}

	if cc.body, err = getBody(p.c.File, p.c.Body); err != nil {
		return
	}

	if cc.tlsConfig, err = getTlsConfig(p.c.Cert, p.c.Key, p.c.Insecure); err != nil {
		return
	}

	var newClient func(clientConfig) (client, error)
	if p.c.Http1 || p.c.Http2 {
		newClient = newHttpClient
	} else {
		newClient = newFasthttpClient
	}

	p.client, err = newClient(cc)

	return
}

func getBody(filename, body string) ([]byte, error) {
	if filename == "" {
		return []byte(body), nil
	}

	return ioutil.ReadFile(filename)
}

func getTlsConfig(cert, key string, insecure bool) (c *tls.Config, err error) {
	var certs []tls.Certificate
	if certs, err = readClientCert(cert, key); err != nil {
		return
	}
	c = &tls.Config{
		Certificates:       certs,
		InsecureSkipVerify: insecure,
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
