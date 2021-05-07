package pit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_nopeLimiter_allow(t *testing.T) {
	t.Parallel()

	var nope nopeLimiter
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			assert.True(t, nope.allow())
		}()
	}
	wg.Wait()
}

func Test_tokenLimiter_allow(t *testing.T) {
	t.Parallel()

	qps, oneTokenDuration := 10, time.Second/10
	lim := newTokenLimiter(qps)
	lim.token = 0

	assert.True(t, lim.allow())
	// cover exceed revoke
	time.Sleep(oneTokenDuration * 2)

	var wg sync.WaitGroup
	wg.Add(qps)
	for i := 0; i < qps; i++ {
		go func() {
			defer wg.Done()
			assert.True(t, lim.allow())
		}()
	}
	wg.Wait()

	assert.False(t, lim.allow())
	time.Sleep(oneTokenDuration)
	assert.True(t, lim.allow())
}
