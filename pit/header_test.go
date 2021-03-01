package pit

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_Header_kvs_invalid_header(t *testing.T) {
	testCases := []headers{
		{"foo"},
		{"foo:bar:baz"},
	}
	for _, tc := range testCases {
		t.Run(tc[0], func(t *testing.T) {
			_, err := tc.kvs()
			assert.NotNil(t, err)
		})
	}
}

func Test_Header_kvs(t *testing.T) {
	testCases := []headers{
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
			kvs, err := tc.kvs()
			assert.Nil(t, err)
			assert.Len(t, kvs, 2)
			assert.Equal(t, "foo", kvs[0])
			assert.Equal(t, "bar", kvs[1])
		})
	}
}

func Test_Header_WriteToFasthttp(t *testing.T) {
	var req fasthttp.Request
	var h headers = []string{"foo:bar", "foo:bar", "bar:baz"}
	assert.Nil(t, h.writeToFasthttp(&req))
	want := "GET / HTTP/1.1\r\nFoo: bar\r\nFoo: bar\r\nBar: baz\r\n\r\n"
	assert.Equal(t, want, string(req.Header.Header()))
}

func Test_Header_WriteToHttp(t *testing.T) {
	var req http.Request
	req.Header = make(http.Header)
	var h headers = []string{"foo:bar", "foo:bar", "bar:baz"}
	assert.Nil(t, h.writeToHttp(&req))
	assert.Equal(t, []string{"bar", "bar"}, req.Header.Values("foo"))
	assert.Equal(t, []string{"baz"}, req.Header.Values("bar"))
}
