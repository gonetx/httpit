package pit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	MIMEApplicationJSON = "application/json"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
)

type client interface {
	do() (int, time.Duration, error)
	doOnce() error
}

type clientDoer interface {
	Do(*fasthttp.Request, *fasthttp.Response) error
}

type onceClientDoer interface {
	clientDoer
	DoRedirects(*fasthttp.Request, *fasthttp.Response, int) error
}

type fasthttpClient struct {
	doer           clientDoer
	onceDoer       onceClientDoer
	reqPool        sync.Pool
	bodyStreamPool sync.Pool
	rawReq         *fasthttp.Request
	body           []byte
	stream         bool
	maxRedirects   int
	wc             io.WriteCloser
}

func newFasthttpClient(c *Config) (fc *fasthttpClient, err error) {
	fc = &fasthttpClient{
		maxRedirects: c.getMaxRedirects(),
		rawReq:       fasthttp.AcquireRequest(),
		stream:       c.Stream,
		wc:           defaultWriteCloser{Writer: os.Stdout},
	}

	c.parseArgs()

	if err = c.setReqBasic(fc.rawReq); err != nil {
		return
	}
	if err = c.setReqBody(fc.rawReq); err != nil {
		return
	}
	fc.body = c.body

	if err = c.setReqHeader(fc.rawReq); err != nil {
		return
	}

	if c.tlsConf, err = c.getTlsConfig(); err != nil {
		return
	}

	if c.Debug {
		fc.rawReq.SetConnectionClose()
		fc.onceDoer, err = c.hostClient()
	} else {
		fc.doer, err = c.doer()
	}

	return
}

func (c *fasthttpClient) do() (code int, latency time.Duration, err error) {
	var (
		req  = c.acquireReq()
		resp = fasthttp.AcquireResponse()
	)

	defer func() {
		c.reqPool.Put(req)
		fasthttp.ReleaseResponse(resp)
	}()

	if c.stream {
		bodyStream := c.acquireBodyStream()
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

func (c *fasthttpClient) acquireReq() *fasthttp.Request {
	v := c.reqPool.Get()
	if v == nil {
		req := fasthttp.AcquireRequest()
		c.rawReq.CopyTo(req)
		return req
	}

	return v.(*fasthttp.Request)
}

func (c *fasthttpClient) acquireBodyStream() *bytes.Reader {
	v := c.bodyStreamPool.Get()
	if v == nil {
		return bytes.NewReader(c.body)
	}
	bodyStream := v.(*bytes.Reader)
	bodyStream.Reset(c.body)
	return bodyStream
}

func (c *fasthttpClient) doOnce() (err error) {
	var (
		req  = c.rawReq
		resp = fasthttp.AcquireResponse()
	)

	if c.stream {
		req.SetBodyStream(bytes.NewReader(c.body), -1)
	}

	defer func() {
		if err == nil {
			// output debug info
			// ignore all errors
			msg := fmt.Sprintf("Connected to %s(%v)\r\n\r\n", req.URI().Host(), resp.RemoteAddr())
			_, _ = c.wc.Write([]byte(msg))
			_, _ = req.WriteTo(c.wc)
			_, _ = c.wc.Write([]byte("\n\n"))
			_, _ = resp.WriteTo(c.wc)
			_ = c.wc.Close()
		}
	}()

	if c.maxRedirects > 0 {
		err = c.onceDoer.DoRedirects(c.rawReq, resp, c.maxRedirects)
	} else {
		err = c.onceDoer.Do(c.rawReq, resp)
	}

	return
}

type discardLogger struct{}

func (discardLogger) Printf(_ string, _ ...interface{}) {}

type defaultWriteCloser struct {
	io.Writer
}

func (wc defaultWriteCloser) Close() error {
	return nil
}
