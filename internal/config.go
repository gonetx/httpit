package internal

import (
	"time"
)

type Config struct {
	Connections int
	Count       int
	Duration    time.Duration
	Timeout     time.Duration

	Method           string
	Headers          Headers
	Body             string
	File             string
	Cert             string
	Key              string
	Stream           bool
	DisableKeepAlive bool
	Insecure         bool
	Http1            bool
	Http2            bool
}
