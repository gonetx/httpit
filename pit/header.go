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
		k, v := strings.ToLower(kvs[i]), kvs[i+1]
		switch k {
		case "host":
			req.URI().SetHost(v)
		case "content-type", "user-agent", "content-length", "connection", "transfer-encoding":
			req.Header.Set(k, v)
		default:
			req.Header.Add(k, v)
		}
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
