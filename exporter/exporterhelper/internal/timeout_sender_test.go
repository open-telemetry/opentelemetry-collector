// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

func TestNewDefaultTimeoutConfig(t *testing.T) {
	cfg := NewDefaultTimeoutConfig()
	require.NoError(t, cfg.Validate())
	assert.Equal(t, TimeoutConfig{Timeout: 5 * time.Second}, cfg)
}

func TestInvalidTimeout(t *testing.T) {
	cfg := NewDefaultTimeoutConfig()
	require.NoError(t, cfg.Validate())
	cfg.Timeout = -1
	assert.Error(t, cfg.Validate())
}

func TestNewTimeoutSender(t *testing.T) {
	cfg := TimeoutConfig{Timeout: 5 * time.Second}
	ts := newTimeoutSender(cfg, sender.NewSender(func(ctx context.Context, data int64) error {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		timeout := time.Since(deadline)
		assert.LessOrEqual(t, timeout, 5*time.Second)
		assert.GreaterOrEqual(t, 4*time.Second, timeout)
		assert.Equal(t, int64(7), data)
		return nil
	}))
	require.NoError(t, ts.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, ts.Send(context.Background(), 7))
	require.NoError(t, ts.Shutdown(context.Background()))
}
