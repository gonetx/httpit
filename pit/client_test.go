package pit

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_Fastclient_New(t *testing.T) {
	t.Parallel()

	t.Run("error header", func(t *testing.T) {
		_, err := newFasthttpClient(&Config{Headers: []string{"a"}})
		assert.NotNil(t, err)
	})

	t.Run("error schema", func(t *testing.T) {
		_, err := newFasthttpClient(&Config{Url: "ftp://host"})
		assert.NotNil(t, err)
	})

	t.Run("http proxy", func(t *testing.T) {
		_, err := newFasthttpClient(&Config{
			HttpProxy: "http://proxy",
		})
		assert.Nil(t, err)
	})

	t.Run("socks proxy", func(t *testing.T) {
		_, err := newFasthttpClient(&Config{
			SocksProxy: "socks5://proxy",
		})
		assert.Nil(t, err)
	})

	t.Run("pipeline with host and close connection", func(t *testing.T) {
		_, err := newFasthttpClient(&Config{
			Url:               "https://127.0.0.1:8443",
			Host:              "example.com",
			DisableKeepAlives: true,
			Pipeline:          true,
		})
		assert.Nil(t, err)
	})

	t.Run("debug mode", func(t *testing.T) {
		fc, err := newFasthttpClient(&Config{
			Url:   "https://127.0.0.1:8443",
			Debug: true,
		})
		assert.Nil(t, err)
		assert.True(t, fc.rawReq.ConnectionClose())
		assert.NotNil(t, fc.onceDoer)
	})
}

func Test_Fastclient_Do(t *testing.T) {
	t.Parallel()

	f := &fasthttpClient{
		body:   []byte("body"),
		stream: true,
	}

	f.rawReq = fasthttp.AcquireRequest()
	f.rawReq.SetRequestURI("http://example.com")
	f.rawReq.Header.SetMethod(fasthttp.MethodGet)

	t.Run("error", func(t *testing.T) {
		fakeErr := errors.New("fake error")
		f.doer = errorFakeDoer(fakeErr, nil)
		_, _, err := f.do()
		assert.NotNil(t, err)
	})

	t.Run("success", func(t *testing.T) {
		f.doer = getFakeDoer(400, t)
		for i := 0; i < 5; i++ {
			code, latency, err := f.do()
			assert.Nil(t, err)
			assert.True(t, latency > 0)
			assert.Equal(t, code, 400)
		}
	})
}

func Test_Fastclient_DoRedirects(t *testing.T) {
	t.Parallel()

	f := &fasthttpClient{
		body:   []byte("body"),
		stream: true,
	}

	f.rawReq = fasthttp.AcquireRequest()
	f.rawReq.SetRequestURI("http://example.com")
	f.rawReq.Header.SetMethod(fasthttp.MethodGet)

	t.Run("error", func(t *testing.T) {
		fakeErr := errors.New("fake error")
		f.onceDoer = errorFakeOnceDoer(fakeErr, nil)
		err := f.doOnce()
		assert.NotNil(t, err)
	})

	t.Run("success", func(t *testing.T) {
		var buf bytes.Buffer
		f.wc = defaultWriteCloser{&buf}
		f.maxRedirects = 10
		f.onceDoer = getFakeOnceDoer(10, t)
		err := f.doOnce()
		assert.Nil(t, err)
	})
}

type fakeDoer struct {
	err  error
	code int
	t    *testing.T
}

func errorFakeDoer(err error, t *testing.T) *fakeDoer {
	return &fakeDoer{err, 0, t}
}

func getFakeDoer(code int, t *testing.T) *fakeDoer {
	return &fakeDoer{t: t, code: code}
}

func (d *fakeDoer) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	if d.err != nil {
		return d.err
	}

	assert.Equal(d.t, "body", string(req.Body()))

	time.Sleep(time.Millisecond * 20)

	resp.Header.SetStatusCode(d.code)

	return nil
}

type fakeOnceDoer struct {
	*fakeDoer
	redirects int
}

func errorFakeOnceDoer(err error, t *testing.T) *fakeOnceDoer {
	return &fakeOnceDoer{
		fakeDoer: &fakeDoer{err, 0, t},
	}
}

func getFakeOnceDoer(redirects int, t *testing.T) *fakeOnceDoer {
	return &fakeOnceDoer{
		fakeDoer:  &fakeDoer{t: t, code: 200},
		redirects: redirects,
	}
}

func (d *fakeOnceDoer) DoRedirects(req *fasthttp.Request, resp *fasthttp.Response, maxRedirects int) error {
	if d.err != nil {
		return d.err
	}

	assert.Equal(d.t, "body", string(req.Body()))
	assert.Equal(d.t, maxRedirects, d.redirects)

	time.Sleep(time.Millisecond * 20)

	resp.Header.SetStatusCode(d.code)

	return nil
}

func Test_discard_pipeline_logger(t *testing.T) {
	discardLogger{}.Printf("")
}
