package internal

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_Header_kvs_invalid_header(t *testing.T) {
	testCases := []Headers{
		{"foo"},
		{"foo:bar:baz"},
	}
	for _, tc := range testCases {
		t.Run(tc[0], func(t *testing.T) {
			assert.Panics(t, func() {
				tc.kvs()
			})
		})
	}
}

func Test_Header_kvs(t *testing.T) {
	testCases := []Headers{
		{"foo:bar"},
		{" foo:bar"},
		{" foo:bar "},
		{"foo: bar"},
		{"foo : bar"},
		{" foo: bar"},
		{" foo: bar "},
		{" foo : bar "},
	}

	for _, tc := range testCases {
		t.Run(tc[0], func(t *testing.T) {
			kvs := tc.kvs()
			assert.Len(t, kvs, 2)
			assert.Equal(t, "foo", kvs[0])
			assert.Equal(t, "bar", kvs[1])
		})
	}
}

func Test_Header_WriteToFasthttp(t *testing.T) {
	var req fasthttp.Request
	var h Headers = []string{"foo:bar", "foo:bar", "bar:baz"}
	h.WriteToFasthttp(&req)
	want := "GET / HTTP/1.1\r\nFoo: bar\r\nFoo: bar\r\nBar: baz\r\n\r\n"
	assert.Equal(t, want, string(req.Header.Header()))
}

func Test_Header_WriteToHttp(t *testing.T) {
	var req http.Request
	req.Header = make(http.Header)
	var h Headers = []string{"foo:bar", "foo:bar", "bar:baz"}
	h.WriteToHttp(&req)
	assert.Equal(t, []string{"bar", "bar"}, req.Header.Values("foo"))
	assert.Equal(t, []string{"baz"}, req.Header.Values("bar"))
}
