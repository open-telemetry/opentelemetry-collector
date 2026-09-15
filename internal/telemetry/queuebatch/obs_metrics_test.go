// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNopObsMetrics(t *testing.T) {
	m := &ObsMetrics{}
	m.EnqueueFailure(context.Background(), 1)
	m.EnqueueSize(context.Background(), 1, func() int64 { return 1 })
	require.NoError(t, m.RegisterQueue(func() int64 { return 1 }, func() int64 { return 1 }))
	m.BatchSendSize(context.Background(), 1, func() int64 { return 1 })
	m.InFlight(context.Background(), 1)
	m.Sent(context.Background(), 1)
	m.SendFailure(context.Background(), 1)
	m.Shutdown()
}

func TestConfigWithObsMetrics(t *testing.T) {
	cfg := struct{}{}
	metrics := ObsMetrics{SentFunc: func(context.Context, int64) {}}

	wrapped := ConfigWithObsMetrics(cfg, metrics)
	gotCfg, gotMetrics, ok := ObsMetricsFromConfig(wrapped)
	require.True(t, ok)
	require.Equal(t, cfg, gotCfg)
	require.NotNil(t, gotMetrics.SentFunc)

	gotCfg, _, ok = ObsMetricsFromConfig(cfg)
	require.False(t, ok)
	require.Equal(t, cfg, gotCfg)
	require.Nil(t, ConfigWithObsMetrics(nil, metrics))
}
