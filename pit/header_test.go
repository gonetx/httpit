package pit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_Header_kvs_invalid_header(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	var req fasthttp.Request
	var h headers = []string{"foo:bar", "foo:bar", "bar:baz", "host:example.com"}
	assert.Nil(t, h.writeToFasthttp(&req))
	want := "GET / HTTP/1.1\r\nFoo: bar\r\nFoo: bar\r\nBar: baz\r\n\r\n"
	assert.Equal(t, want, string(req.Header.Header()))
	assert.Equal(t, "example.com", string(req.URI().Host()))
}
