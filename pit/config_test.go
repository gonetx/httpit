package pit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_Config_setReqBasic(t *testing.T) {
	t.Parallel()

	t.Run("unsupported protocol", func(t *testing.T) {
		c, req := configAndReq()
		c.Url = "ftp://uri"
		assert.NotNil(t, c.setReqBasic(req))
	})

	t.Run("https", func(t *testing.T) {
		c, req := configAndReq()
		c.Url = "https://1.1.1.1:8443"
		assert.Nil(t, c.setReqBasic(req))
		assert.True(t, c.isTLS)
		assert.Equal(t, "1.1.1.1:8443", c.addr)
	})

	t.Run("default protocol", func(t *testing.T) {
		c, req := configAndReq()
		c.Url = "http://example.com"
		assert.Nil(t, c.setReqBasic(req))
		assert.False(t, c.isTLS)
		assert.Equal(t, "example.com:80", c.addr)
	})
}

func Test_Config_parseArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		args     []string
		isForm   bool
		isJson   bool
		expected string
	}{
		{"form no value", []string{"foo"}, true, false, "foo"},
		{"form one kv", []string{" foo=bar"}, true, false, "foo=bar"},
		{"form two kv", []string{"foo=bar ", "bar=baz"}, true, false, "foo=bar&bar=baz"},
		{"form escape kv", []string{"foo=bar=baz"}, true, false, "foo=bar%3Dbaz"},
		{"json no value", []string{"foo:="}, false, true, `{"foo":""}`},
		{"json bool true value", []string{" foo:=true"}, false, true, `{"foo":true}`},
		{"json bool false value", []string{"foo:= False"}, false, true, `{"foo":False}`},
		{"json int value", []string{"foo:=1 "}, false, true, `{"foo":1}`},
		{"json float value", []string{"foo:=1.1"}, false, true, `{"foo":1.1}`},
		{"json array value", []string{"foo :=[1]"}, false, true, `{"foo":[1]}`},
		{"json object value", []string{`foo:={"bar":"baz"}`}, false, true, `{"foo":{"bar":"baz"}}`},
		{"json string value", []string{`foo:= baz`}, false, true, `{"foo":"baz"}`},
		{"json invalid bool tru value", []string{"foo:=tru"}, false, true, `{"foo":"tru"}`},
		{"json invalid bool false value", []string{"foo:=fals"}, false, true, `{"foo":"fals"}`},
		{"json invalid int value", []string{"foo:=1aa"}, false, true, `{"foo":"1aa"}`},
		{"json invalid float value", []string{"foo:=1.1.1"}, false, true, `{"foo":"1.1.1"}`},
		{"json two kv", []string{"foo:=true", "bar:=1"}, false, true, `{"foo":true,"bar":1}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := configAndReq()
			c.Args = tc.args
			c.parseArgs()
			assert.Equal(t, tc.isForm, c.Form)
			assert.Equal(t, tc.isJson, c.JSON)
			assert.Equal(t, tc.expected, string(c.body))
		})
	}
}

func Test_Config_setReqBody(t *testing.T) {
	t.Parallel()

	t.Run("from file", func(t *testing.T) {
		t.Run("error", func(t *testing.T) {
			c, req := configAndReq()
			c.File = "non-exist"
			assert.NotNil(t, c.setReqBody(req))
			assert.Equal(t, string(c.body), "")
		})

		t.Run("success", func(t *testing.T) {
			c, req := configAndReq()
			c.File = "testdata/ssl.pem"
			assert.Nil(t, c.setReqBody(req))
			assert.NotNil(t, string(c.body), "")
		})
	})

	t.Run("from flag body", func(t *testing.T) {
		c, req := configAndReq()
		c.Body = "body"
		assert.Nil(t, c.setReqBody(req))
		assert.Equal(t, string(c.body), c.Body)
	})

	t.Run("not stream", func(t *testing.T) {
		c, req := configAndReq()
		c.Body = "body"
		assert.Nil(t, c.setReqBody(req))
		assert.Equal(t, c.body, req.Body())
	})
}

func Test_Config_setReqHeader(t *testing.T) {
	t.Parallel()

	t.Run("append flag header", func(t *testing.T) {
		t.Run("error", func(t *testing.T) {
			c, req := configAndReq()
			c.Headers = []string{"k1"}
			assert.NotNil(t, c.setReqHeader(req))
		})
		t.Run("success", func(t *testing.T) {
			c, req := configAndReq()
			c.Headers = []string{"k1:v1"}
			assert.Nil(t, c.setReqHeader(req))
			assert.Equal(t, "v1", string(req.Header.Peek("K1")))
		})
	})

	t.Run("disable keep alive", func(t *testing.T) {
		c, req := configAndReq()
		c.DisableKeepAlives = true
		assert.Nil(t, c.setReqHeader(req))
		assert.True(t, req.Header.ConnectionClose())
	})

	t.Run("override uri host", func(t *testing.T) {
		c, req := configAndReq()
		c.Host = "example.com"
		assert.Nil(t, c.setReqHeader(req))
		assert.Equal(t, c.Host, string(req.URI().Host()))
	})

	t.Run("json", func(t *testing.T) {
		c, req := configAndReq()
		c.JSON = true
		assert.Nil(t, c.setReqHeader(req))
		assert.Equal(t, MIMEApplicationJSON, string(req.Header.ContentType()))
	})

	t.Run("form", func(t *testing.T) {
		c, req := configAndReq()
		c.Form = true
		assert.Nil(t, c.setReqHeader(req))
		assert.Equal(t, MIMEApplicationForm, string(req.Header.ContentType()))
	})
}

func configAndReq() (*Config, *fasthttp.Request) {
	return &Config{}, &fasthttp.Request{}
}

func Test_getMaxRedirects(t *testing.T) {
	t.Parallel()

	t.Run("non follow", func(t *testing.T) {
		c := &Config{}

		assert.Equal(t, 0, c.getMaxRedirects())
	})

	t.Run("use default value", func(t *testing.T) {
		c := &Config{Follow: true}

		assert.Equal(t, defaultMaxRedirects, c.getMaxRedirects())
	})

	t.Run("use custom value", func(t *testing.T) {
		c := &Config{Follow: true, MaxRedirects: 10}

		assert.Equal(t, 10, c.getMaxRedirects())
	})
}

func Test_readClientCert(t *testing.T) {
	t.Parallel()

	cert, err := readClientCert("testdata/ssl.pem", "testdata/ssl.key")
	assert.Nil(t, err)
	assert.Len(t, cert, 1)
}

func Test_addMissingPort(t *testing.T) {
	t.Parallel()

	t.Run("return directly", func(t *testing.T) {
		addr := "127.0.0.1:8080"
		assert.Equal(t, addr, addMissingPort(addr, false))
	})

	t.Run("append 443", func(t *testing.T) {
		addr := "127.0.0.1"
		assert.Equal(t, addr+":443", addMissingPort(addr, true))
	})
}

func Test_Config_hostClient(t *testing.T) {
	c := &Config{
		Http2: true,
		addr:  "127.0.0.1:8443",
	}

	hc, err := c.hostClient()
	assert.Nil(t, hc)
	assert.EqualError(t, err, "127.0.0.1:8443 doesn't support http/2\n")
}
