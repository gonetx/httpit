package pit

import (
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"
)

type headers []string

func (h headers) writeToFasthttp(req *fasthttp.Request) error {
	kvs, err := h.kvs()
	if err != nil {
		return err
	}
	for i := 0; i < len(kvs); i += 2 {
		k, v := kvs[i], kvs[i+1]
		req.Header.Add(k, v)
	}
	return nil
}

func (h headers) kvs() ([]string, error) {
	list := make([]string, 0, len(h)*2)
	for _, header := range h {
		kv := strings.Split(header, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("failed to parse request header %s", header)
		}
		list = append(list, strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
	return list, nil
}
