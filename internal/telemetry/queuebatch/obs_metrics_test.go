// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
)

func TestNopObsMetrics(t *testing.T) {
	m := &ObsMetrics{}
	require.False(t, m.ShouldRecord(context.Background(), MetricEnqueueSizeBytes))
	m.RecordInt(context.Background(), MetricSent, 1)
	require.NoError(t, m.RegisterInt(MetricQueueSize, func() int64 { return 1 }))
	m.Shutdown()
}

func TestObsMetrics(t *testing.T) {
	calls := 0
	m := ObsMetrics{
		ShouldRecordFunc: func(context.Context, Metric) bool {
			calls++
			return true
		},
		RecordIntFunc: func(context.Context, Metric, int64, ...metric.AddOption) {
			calls++
		},
		RegisterIntFunc: func(_ Metric, value func() int64) error {
			require.Equal(t, int64(1), value())
			calls++
			return nil
		},
		ShutdownFunc: func() { calls++ },
	}

	ctx := context.Background()
	require.True(t, m.ShouldRecord(ctx, MetricEnqueueSizeBytes))
	m.RecordInt(ctx, MetricSent, 1)
	require.NoError(t, m.RegisterInt(MetricQueueSize, func() int64 { return 1 }))
	m.Shutdown()
	require.Equal(t, 4, calls)
}

func TestConfigWithObsMetrics(t *testing.T) {
	cfg := struct{}{}
	metrics := ObsMetrics{
		RecordIntFunc: func(context.Context, Metric, int64, ...metric.AddOption) {},
	}

	wrapped := ConfigWithObsMetrics(cfg, metrics)
	gotCfg, gotMetrics, ok := ObsMetricsFromConfig(wrapped)
	require.True(t, ok)
	require.Equal(t, cfg, gotCfg)
	require.NotNil(t, gotMetrics.RecordIntFunc)

	gotCfg, _, ok = ObsMetricsFromConfig(cfg)
	require.False(t, ok)
	require.Equal(t, cfg, gotCfg)
	require.Nil(t, ConfigWithObsMetrics(nil, metrics))
}
