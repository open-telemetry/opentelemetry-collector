// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtimeout"
	"go.opentelemetry.io/collector/exporter/internal"
)

func TestNewDefaultTimeoutConfig(t *testing.T) {
	cfg := configtimeout.NewDefaultConfig()
	require.NoError(t, cfg.Validate())
	assert.Equal(t, configtimeout.TimeoutConfig{Timeout: 5 * time.Second}, cfg)
}

func TestInvalidTimeout(t *testing.T) {
	cfg := configtimeout.NewDefaultConfig()
	require.NoError(t, cfg.Validate())
	cfg.Timeout = -1
	assert.Error(t, cfg.Validate())
}

const expectCtxVal = "testval"

type expectCtxKey struct{}

func newTestContext() context.Context {
	return context.WithValue(context.Background(), expectCtxKey{}, expectCtxVal)
}

type testTimeoutRequest struct {
	t       *testing.T
	err     error
	calls   int
	lastCtx context.Context
}

func (s *testTimeoutRequest) Export(ctx context.Context) error {
	s.calls++
	s.lastCtx = ctx
	require.Equal(s.t, expectCtxVal, ctx.Value(expectCtxKey{}))
	if s.err != nil {
		return s.err
	}
	return ctx.Err()
}

func (s *testTimeoutRequest) OnError(error) internal.Request {
	return s
}

func (s *testTimeoutRequest) ItemsCount() int {
	return 11
}

func TestTimeoutSustain(t *testing.T) {
	for _, pol := range []configtimeout.Policy{
		configtimeout.PolicySustain, // Test w/ explicit name
		"",                          // Test an unset value
	} {
		ts := TimeoutSender{
			cfg: configtimeout.TimeoutConfig{
				Timeout:            5 * time.Second,
				ShortTimeoutPolicy: pol,
			},
		}
		ctx, cancel := context.WithTimeout(newTestContext(), 0)
		defer cancel()

		// The context is already canceled, but w/ a Sustain policy
		// it will call Export() anyway.
		tt1 := &testTimeoutRequest{t: t, err: fmt.Errorf("return this")}
		require.Equal(t, tt1.err, ts.Send(ctx, tt1))
		require.Equal(t, 1, tt1.calls)

		// mockRequest returns ctx.Err() in this case.
		tt2 := &testTimeoutRequest{t: t}
		require.ErrorIs(t,
			ts.Send(ctx, tt2),
			context.DeadlineExceeded,
		)
		require.Equal(t, 1, tt2.calls)
	}
}

func TestTimeoutIgnoreSuccess(t *testing.T) {
	ts := TimeoutSender{
		cfg: configtimeout.TimeoutConfig{
			Timeout:            5 * time.Second,
			ShortTimeoutPolicy: configtimeout.PolicyIgnore,
		},
	}
	ctx, cancel := context.WithTimeout(newTestContext(), 0)
	defer cancel()

	// The context is already canceled, but w/ an Ignore policy
	// itwill call Export() anyway using a context free of the deadline.
	tt1 := &testTimeoutRequest{t: t}
	require.NoError(t, ts.Send(ctx, tt1))
	require.Equal(t, 1, tt1.calls)
}

func TestTimeoutIgnoreCanceled(t *testing.T) {
	ts := TimeoutSender{
		cfg: configtimeout.TimeoutConfig{
			Timeout:            5 * time.Second,
			ShortTimeoutPolicy: configtimeout.PolicyIgnore,
		},
	}
	ctx, cancel := context.WithCancel(newTestContext())
	cancel()

	tt1 := &testTimeoutRequest{t: t}
	require.ErrorIs(t,
		ts.Send(ctx, tt1),
		context.Canceled,
	)
	require.Equal(t, 1, tt1.calls)
}

func TestTimeoutZeroNoDeadline(t *testing.T) {
	ts := TimeoutSender{}
	ctx := newTestContext()

	tt1 := &testTimeoutRequest{t: t}
	require.NoError(t, ts.Send(ctx, tt1))
	require.Equal(t, 1, tt1.calls)
	_, ok := tt1.lastCtx.Deadline()
	require.False(t, ok, "export context has no deadline")
}

func TestTimeoutZeroWithDeadline(t *testing.T) {
	ts := TimeoutSender{}
	ctx, cancel := context.WithTimeout(newTestContext(), time.Hour)
	defer cancel()

	tt1 := &testTimeoutRequest{t: t}
	require.NoError(t, ts.Send(ctx, tt1))
	require.Equal(t, 1, tt1.calls)
	_, ok := tt1.lastCtx.Deadline()
	require.True(t, ok, "export context has deadline")
}

func TestTimeoutZeroIgnoreDeadline(t *testing.T) {
	ts := TimeoutSender{
		cfg: configtimeout.TimeoutConfig{
			Timeout:            0,
			ShortTimeoutPolicy: configtimeout.PolicyIgnore,
		},
	}
	ctx, cancel := context.WithTimeout(newTestContext(), time.Hour)
	defer cancel()

	tt1 := &testTimeoutRequest{t: t}
	require.NoError(t, ts.Send(ctx, tt1))
	require.Equal(t, 1, tt1.calls)
	_, ok := tt1.lastCtx.Deadline()
	require.False(t, ok, "export context has no deadline")
}

func TestTimeoutAbort(t *testing.T) {
	ts := TimeoutSender{
		cfg: configtimeout.TimeoutConfig{
			Timeout:            5 * time.Second,
			ShortTimeoutPolicy: configtimeout.PolicyAbort,
		},
	}
	vctx := newTestContext()
	ctx, cancel := context.WithTimeout(vctx, 0)
	defer cancel()

	// The context is already canceled, but w/ an Ignore policy
	// itwill call Export() anyway using a context free of the deadline.
	tt1 := &testTimeoutRequest{t: t}
	err := ts.Send(ctx, tt1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "request will be cancelled in")
}
