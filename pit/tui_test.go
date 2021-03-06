package pit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_writeProcessBar(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.count = 1
	tt.reqs = 2
	tt.writeProcessBar()
	assert.Contains(t, tt.buf.String(), "100%")
}

func Test_writeTotalRequest(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.count = 3
	tt.reqs = 2
	tt.writeTotalRequest()
	assert.Contains(t, tt.buf.String(), "2/3")
}

func Test_writeElapsed(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.duration = time.Second
	tt.elapsed = int64(time.Second * 2)
	tt.writeElapsed()
	assert.Contains(t, tt.buf.String(), "1.00/1.00")
}

func Test_writeThroughput(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.throughput = 1001
	tt.elapsed = int64(time.Second)
	tt.writeThroughput()
	assert.Contains(t, tt.buf.String(), "1.00 KB/s")
}

func Test_writeErrors(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.errs["custom-error"] = 1
	tt.writeErrors()
	assert.Contains(t, tt.buf.String(), "custom-error")
	assert.Contains(t, tt.buf.String(), "1")
}
