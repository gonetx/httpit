package pit

import (
	"errors"
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stretchr/testify/assert"
)

func Test_Pit_Run(t *testing.T) {
	t.Parallel()

	t.Run("missing url", func(t *testing.T) {
		p := New(Config{})
		assert.NotNil(t, p.Run())
	})

	t.Run("do once in debug mode", func(t *testing.T) {
		p := New(Config{Url: "url", Debug: true})
		p.client = newFakeClient()
		assert.Nil(t, p.Run())
	})

	t.Run("success", func(t *testing.T) {
		p := New(Config{Url: "url"})
		p.tui.initCmd = func() tea.Msg {
			return tea.Quit()
		}
		p.tui.w = ioutil.Discard
		p.tui.r = os.Stdin

		p.client = newFakeClient()

		assert.Nil(t, p.Run())
	})
}

func Test_Pit_Init(t *testing.T) {
	t.Parallel()

	url := "url"

	t.Run("missing url", func(t *testing.T) {
		p := New(Config{})
		assert.NotNil(t, p.init())
	})

	t.Run("no file", func(t *testing.T) {
		p := New(Config{Url: url, File: "not-exist"})
		assert.NotNil(t, p.init())
	})

	t.Run("no cert", func(t *testing.T) {
		p := New(Config{Url: url, Cert: "not-cert"})
		assert.NotNil(t, p.init())
	})

	t.Run("success", func(t *testing.T) {
		p := New(Config{Url: url})
		assert.Nil(t, p.init())
	})
}

func Test_Pit_Internal_Run(t *testing.T) {
	t.Parallel()

	p := New(Config{})
	p.c.Connections = 2
	p.c.Count = 2
	p.client = newFakeClient()
	assert.Equal(t, done, p.run().(int))
}

func Test_Pit_Statistic(t *testing.T) {
	t.Parallel()

	t.Run("already done", func(t *testing.T) {
		p := New(Config{})
		p.done = true
		p.statistic(200, 0, nil)
		assert.Equal(t, int64(0), p.roundReqs)
	})

	t.Run("got error", func(t *testing.T) {
		p := New(Config{})
		p.statistic(200, time.Millisecond, errors.New(""))
		assert.Equal(t, 1, len(p.tui.errs))
	})

	t.Run("reach count", func(t *testing.T) {
		p := New(Config{})
		p.c.Count = 1
		p.statistic(200, time.Millisecond, nil)
		assert.Equal(t, int64(1), p.tui.code2xx)
		assert.Equal(t, 1, len(p.tui.latencies))
		assert.True(t, p.done)
	})

	t.Run("reach duration", func(t *testing.T) {
		p := New(Config{})
		p.startTime = time.Now().Add(-time.Second)
		p.c.Duration = time.Millisecond * 10
		p.statistic(200, time.Millisecond, nil)
		assert.Equal(t, int64(1), p.tui.code2xx)
		assert.Equal(t, 1, len(p.tui.latencies))
		assert.True(t, p.done)
	})
}

func Test_addMissingSchemaAndHost(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		url    string
		target string
	}{
		{":3000", "http://127.0.0.1:3000"},
		{"127.0.0.1:3000", "http://127.0.0.1:3000"},
		{"example.com", "http://example.com"},
		{"http://example.com", "http://example.com"},
		{"https://example.com", "https://example.com"},
		{"ftp://example.com", "ftp://example.com"},
		{"://example.com", "://example.com"},
		{"//example.com", "//example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			assert.Equal(t, tc.target, addMissingSchemaAndHost(tc.url))
		})
	}
}

type fakeClient struct {
	err   error
	count int64
}

func newFakeClient(err ...error) *fakeClient {
	var e error
	if len(err) > 0 {
		e = err[0]
	}
	return &fakeClient{err: e}
}

func (fc *fakeClient) do() (int, time.Duration, error) {
	atomic.AddInt64(&fc.count, 1)
	return 0, 0, nil
}

func (fc *fakeClient) doOnce() error {
	return fc.err
}
