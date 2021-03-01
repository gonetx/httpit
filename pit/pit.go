package pit

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	defaultConnections = 128
	defaultDuration    = time.Second * 10
	defaultTimeout     = time.Second * 3
)

type Pit struct {
	c *Config
	client
	throughput int64

	wg        sync.WaitGroup
	mut       sync.Mutex
	start     time.Time
	elapsed   time.Duration
	totalReqs int64
	reqs      int64
	errs      map[string]int
	done      bool
	doneChan  chan struct{}

	// HTTP codes
	code1xx    uint64
	code2xx    uint64
	code3xx    uint64
	code4xx    uint64
	code5xx    uint64
	codeOthers uint64

	latencies []int64
	rps       []float64
}

func New(c Config) *Pit {
	p := &Pit{
		c:        &c,
		doneChan: make(chan struct{}),
		errs:     make(map[string]int),
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

	p.print()

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
		p.errs[err.Error()]++
	} else {
		p.reqs++
		p.totalReqs++
		p.handleCode(code)
		p.latencies = append(p.latencies, latency.Microseconds())
	}

	elapsed := time.Since(p.start)
	// reached count
	if p.c.Count > 0 && p.totalReqs == int64(p.c.Count) {
		p.rps = append(p.rps, float64(p.reqs)/elapsed.Seconds())

		p.done = true
		// notify workers to stop
		close(p.doneChan)
		return
	}

	// one round is over
	if elapsed >= interval {
		p.rps = append(p.rps, float64(p.reqs)/elapsed.Seconds())

		p.elapsed += elapsed
		p.start = time.Now()
		p.reqs = 0
	}

	if p.c.Count <= 0 && p.elapsed >= p.c.Duration {
		p.done = true
		// notify workers to stop
		close(p.doneChan)
	}
}

func (p *Pit) handleCode(code int) {
	switch code / 100 {
	case 1:
		p.code1xx++
	case 2:
		p.code2xx++
	case 3:
		p.code3xx++
	case 4:
		p.code4xx++
	case 5:
		p.code5xx++
	default:
		p.codeOthers++
	}
}

const temp = `Total requests: %d
Elapsed: %.2fs
Statistics        Avg      Stdev        Max
  Reqs/sec       %.2f     %.2f     %.2f
  Latency      %.2fms   %.2fms     %.2fms
  HTTP codes:
    1xx - %d, 2xx - %d, 3xx - %d, 4xx - %d, 5xx - %d
    others - %d
  Throughput: %s
`

func (p *Pit) print() {
	rpsAvg, rpsStdev, rpsMax := rpsResult(p.rps)
	latencyAvg, latencyStdev, latencyMax := latencyResult(p.latencies)

	output := fmt.Sprintf(temp, p.totalReqs, p.elapsed.Seconds(),
		rpsAvg, rpsStdev, rpsMax,
		latencyAvg, latencyStdev, latencyMax,
		p.code1xx, p.code2xx, p.code3xx, p.code4xx, p.code5xx,
		p.codeOthers,
		formatThroughput(float64(p.throughput)/p.elapsed.Seconds()),
	)

	if len(p.errs) > 0 {
		output += errorResult(p.errs)
	}

	fmt.Print(output)
}

func rpsResult(rps []float64) (avg float64, stdev float64, max float64) {
	var sum, sum2 float64
	for _, r := range rps {
		sum += r
		if r > max {
			max = r
		}
	}

	avg = sum / float64(len(rps))

	var diff float64
	for _, r := range rps {
		diff = avg - r
		sum2 += diff * diff
	}

	stdev = math.Sqrt(sum2 / float64(len(rps)-1))

	return
}

func latencyResult(latencies []int64) (avg float64, stdev float64, max float64) {
	var sum float64
	for _, latency := range latencies {
		// us -> ms
		r := float64(latency) / 1000
		sum += r
		if r > max {
			max = r
		}
	}

	avg = sum / float64(len(latencies))

	var diff, sum2 float64
	for _, latency := range latencies {
		// us -> ms
		r := float64(latency) / 1000
		diff = avg - r
		sum2 += diff * diff
	}

	stdev = math.Sqrt(sum2 / float64(len(latencies)-1))

	return
}

func formatThroughput(throughput float64) string {
	switch {
	case throughput < 1e3:
		return fmt.Sprintf("%.2f B/s", throughput)
	case throughput < 1e6:
		return fmt.Sprintf("%.2f KB/s", throughput/1e3)
	case throughput < 1e9:
		return fmt.Sprintf("%.2f MB/s", throughput/1e6)
	default:
		return fmt.Sprintf("%.2f GB/s", throughput/1e12)
	}
}

func errorResult(errs map[string]int) string {
	var sb strings.Builder
	sb.WriteString("  Errors\n")
	for err, count := range errs {
		sb.WriteString(fmt.Sprintf("    %s: %d\n", err, count))
	}

	return sb.String()
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
