package pit

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_Fastclient_New(t *testing.T) {
	t.Parallel()

	t.Run("error header", func(t *testing.T) {
		_, err := newFasthttpClient(clientConfig{headers: []string{"a"}})
		assert.NotNil(t, err)
	})

	t.Run("error schema", func(t *testing.T) {
		_, err := newFasthttpClient(clientConfig{url: "ftp://host"})
		assert.NotNil(t, err)
	})

	t.Run("http proxy", func(t *testing.T) {
		_, err := newFasthttpClient(clientConfig{
			httpProxy: "http://proxy",
		})
		assert.Nil(t, err)
	})

	t.Run("socks proxy", func(t *testing.T) {
		_, err := newFasthttpClient(clientConfig{
			socksProxy: "socks5://proxy",
		})
		assert.Nil(t, err)
	})

	t.Run("pipeline with host and close connection", func(t *testing.T) {
		_, err := newFasthttpClient(clientConfig{
			url:               "https://127.0.0.1:8443",
			host:              "example.com",
			disableKeepAlives: true,
			pipeline:          true,
		})
		assert.Nil(t, err)
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

	f.initPool()

	t.Run("error", func(t *testing.T) {
		fakeErr := errors.New("fake error")
		f.doer = errorFakeDoer(fakeErr, nil)
		_, _, err := f.do()
		assert.NotNil(t, err)
	})

	t.Run("success", func(t *testing.T) {
		f.doer = getFakeDoer(400, t)
		code, latency, err := f.do()
		assert.Nil(t, err)
		assert.True(t, latency > 0)
		assert.Equal(t, code, 400)
	})
}

func Test_addMissingPort(t *testing.T) {
	t.Parallel()

	addr := "127.0.0.1:8080"
	assert.Equal(t, addr, addMissingPort(addr, false))
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

func Test_discard_pipeline_logger(t *testing.T) {
	discardLogger{}.Printf("")
}
