package pit

import (
	"time"
)

type Config struct {
	Connections int
	Count       int
	Duration    time.Duration
	Timeout     time.Duration

	Method            string
	Headers           []string
	Host              string
	Body              string
	File              string
	Cert              string
	Key               string
	Stream            bool
	DisableKeepAlives bool
	Insecure          bool
}
