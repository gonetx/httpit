package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
)

type Headers []string

func (h Headers) WriteToFasthttp(req *fasthttp.Request) {
	kvs := h.kvs()
	for i := 0; i < len(kvs); i += 2 {
		k, v := kvs[i], kvs[i+1]
		req.Header.Add(k, v)
	}
}

func (h Headers) WriteToHttp(req *http.Request) {
	kvs := h.kvs()
	for i := 0; i < len(kvs); i += 2 {
		k, v := kvs[i], kvs[i+1]
		req.Header.Add(k, v)
	}
}

func (h Headers) kvs() []string {
	list := make([]string, 0, len(h)*2)
	for _, header := range h {
		kv := strings.Split(header, ":")
		if len(kv) != 2 {
			panic(fmt.Sprintf("failed to parse request header %s", header))
		}
		list = append(list, strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
	return list
}
