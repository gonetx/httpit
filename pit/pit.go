package pit

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Version of current httpit
const Version = "0.3.2"

const (
	defaultConnections  = 128
	defaultDuration     = time.Second * 10
	defaultTimeout      = time.Second * 3
	defaultMaxRedirects = 30
)

// Pit denotes httpit application
type Pit struct {
	c *Config
	client
	wg sync.WaitGroup

	mut       sync.Mutex
	startTime time.Time
	roundReqs int64
	done      bool
	doneChan  chan struct{}
	*tui
}

// New create a Pit instance with specific Config
func New(c Config) *Pit {
	p := &Pit{
		c:        &c,
		doneChan: make(chan struct{}),
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

	if p.c.MaxRedirects <= 0 {
		p.c.MaxRedirects = defaultMaxRedirects
	}

	p.tui = newTui()
	p.tui.count = p.c.Count
	p.tui.duration = p.c.Duration
	p.tui.connections = p.c.Connections
	p.tui.throughput = &p.c.throughput
	p.initCmd = p.run

	return p
}

// Run starts benchmarking
func (p *Pit) Run() (err error) {
	if err = p.init(); err != nil {
		return
	}

	if p.c.Debug {
		return p.doOnce()
	}

	return p.tui.start()
}

func (p *Pit) init() (err error) {
	if p.c.Url == "" {
		return errors.New("missing url")
	}

	// :3000 => http://127.0.0.1
	// example.com => http://example.com
	p.c.Url = addMissingSchemaAndHost(p.c.Url)
	p.tui.url = p.c.Url

	if p.client == nil {
		p.client, err = newFasthttpClient(p.c)
	}

	return
}

func addMissingSchemaAndHost(url string) string {
	if !strings.HasPrefix(url, "://") && strings.HasPrefix(url, ":") {
		// :3000 => http://localhost:3000
		return "http://localhost" + url
	}
	if strings.Index(url, "://") == -1 && len(url) >= 2 {
		if url[0] == '/' && url[1] != '/' {
			// /foo => http://localhost/foo
			return "http://localhost" + url
		}
		if url[0] != '/' && url[1] != '/' {
			// example.com => http://example.com
			return "http://" + url
		}
	}
	return url
}

func (p *Pit) run() tea.Msg {
	p.startTime = time.Now()
	n := p.c.Connections
	p.wg.Add(n)
	for i := 0; i < n; i++ {
		go p.worker()
	}
	// wait for all workers stop
	p.wg.Wait()

	return done
}

func (p *Pit) worker() {
	for {
		select {
		case <-p.doneChan:
			p.wg.Done()
			return
		default:
			p.statistic(p.do())
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
		p.appendError(err)
	} else {
		p.roundReqs++
		atomic.AddInt64(&p.reqs, 1)
		p.appendCode(code)
		p.appendLatency(latency)
	}

	elapsed := time.Since(p.startTime)
	// reached count
	if p.c.Count > 0 && atomic.LoadInt64(&p.reqs) == int64(p.c.Count) {
		p.appendRps(float64(p.roundReqs) / elapsed.Seconds())
		p.done = true
		// notify workers to stop
		close(p.doneChan)
		return
	}

	// one round is over
	if elapsed >= interval {
		p.appendRps(float64(p.roundReqs) / elapsed.Seconds())

		atomic.AddInt64(&p.elapsed, int64(elapsed))

		p.startTime = time.Now()
		p.roundReqs = 0
	}

	if p.c.Count <= 0 && atomic.LoadInt64(&p.elapsed) >= int64(p.c.Duration) {
		p.done = true
		// notify workers to stop
		close(p.doneChan)
	}
}
