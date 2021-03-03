package pit

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/gonetx/httpit/gui"
)

const (
	defaultConnections = 128
	defaultDuration    = time.Second * 10
	defaultTimeout     = time.Second * 3
)

type Pit struct {
	c *Config
	client
	wg sync.WaitGroup

	mut      sync.Mutex
	start    time.Time
	reqs     int64
	done     bool
	doneChan chan struct{}

	r *gui.Result
}

func New(c Config) *Pit {
	p := &Pit{
		c:        &c,
		doneChan: make(chan struct{}),
		r:        gui.NewResult(os.Stdout),
	}

	if p.c.Connections <= 0 {
		p.c.Connections = defaultConnections
	}

	if p.c.Duration <= 0 {
		p.c.Duration = defaultDuration
	}

	if p.c.Timeout <= 0 {
		p.c.Timeout = defaultTimeout
	}

	return p
}

func (p *Pit) Run(url string) (err error) {
	if err = p.init(url); err != nil {
		return
	}

	p.start = time.Now()
	n := p.c.Connections
	p.wg.Add(n)
	for i := 0; i < n; i++ {
		go p.worker(i)
	}
	// wait for all workers stop
	p.wg.Wait()

	p.r.Print()

	return
}

func (p *Pit) init(url string) (err error) {
	if url == "" {
		return errors.New("missing url")
	}

	cc := clientConfig{
		method:            p.c.Method,
		url:               url,
		headers:           p.c.Headers,
		host:              p.c.Host,
		stream:            p.c.Stream,
		maxConns:          p.c.Connections,
		timeout:           p.c.Timeout,
		disableKeepAlives: p.c.DisableKeepAlives,
		throughput:        p.r.Throughput(),
	}

	if cc.body, err = getBody(p.c.File, p.c.Body); err != nil {
		return
	}

	if cc.tlsConfig, err = getTlsConfig(p.c.Cert, p.c.Key, p.c.Insecure); err != nil {
		return
	}

	p.client, err = newFasthttpClient(cc)

	return
}

func (p *Pit) worker(i int) {
	for {
		select {
		case <-p.doneChan:
			p.wg.Done()
			return
		default:
			p.statistic(p.do(i))
		}
	}
}

const interval = time.Millisecond * 10

func (p *Pit) statistic(code int, latency time.Duration, err error) {
	p.mut.Lock()
	defer p.mut.Unlock()
	if p.done {
		return
	}

	if err != nil {
		p.r.AppendError(err)
	} else {
		p.reqs++
		p.r.IncreaseReq()
		p.r.AppendCode(code)
		p.r.AppendLatency(latency)
	}

	elapsed := time.Since(p.start)
	// reached count
	if p.c.Count > 0 && p.r.TotalReqs() == int64(p.c.Count) {
		p.r.AppendRps(float64(p.reqs) / elapsed.Seconds())
		p.done = true
		// notify workers to stop
		close(p.doneChan)
		return
	}

	// one round is over
	if elapsed >= interval {
		p.r.AppendRps(float64(p.reqs) / elapsed.Seconds())

		p.r.AddElapsed(elapsed)
		p.start = time.Now()
		p.reqs = 0
	}

	if p.c.Count <= 0 && p.r.Elapsed() >= p.c.Duration {
		p.done = true
		// notify workers to stop
		close(p.doneChan)
	}
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
