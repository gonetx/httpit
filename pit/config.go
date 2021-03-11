package pit

import (
	"time"
)

type Config struct {
	Connections        int
	Count              int
	Duration           time.Duration
	Timeout            time.Duration
	Method             string
	Headers            []string
	Host               string
	DisableKeepAlives  bool
	Body               string
	File               string
	Stream             bool
	JSON               bool
	Form               bool
	MultipartForm      bool
	MultipartFormFiles []string
	Boundary           string
	Insecure           bool
	Cert               string
	Key                string
	HttpProxy          string
	SocksProxy         string
	Pipeline           bool
	Follow             bool
	MaxRedirects       int
	Output             string
	Quite              bool
	Debug              bool
}
