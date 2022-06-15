package redis_state

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest-cli/pkg/execution/state/testharness"
)

func TestStateHarness(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()
	m := New(WithConnectOpts(redis.Options{
		Addr: s.Addr(),
	}))
	testharness.CheckState(t, m)
}
