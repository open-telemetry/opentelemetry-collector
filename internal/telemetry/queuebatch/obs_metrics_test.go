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
	m.EnqueueFailure(context.Background(), 1)
	m.EnqueueSize(context.Background(), 1, func() int64 { return 1 })
	require.NoError(t, m.RegisterQueue(func() int64 { return 1 }, func() int64 { return 1 }))
	m.BatchSendSize(context.Background(), 1, func() int64 { return 1 })
	m.InFlight(context.Background(), 1)
	m.Sent(context.Background(), 1)
	m.SendFailure(context.Background(), 1)
	m.Shutdown()
}

func TestObsMetrics(t *testing.T) {
	calls := 0
	call := func(context.Context, int64) { calls++ }
	sizeCall := func(context.Context, int64, func() int64) { calls++ }
	m := ObsMetrics{
		QueueMetrics: QueueMetrics{
			EnqueueFailureFunc: call,
			EnqueueSizeFunc:    sizeCall,
			RegisterQueueFunc: func(size, capacity func() int64) error {
				require.Equal(t, int64(1), size())
				require.Equal(t, int64(2), capacity())
				calls++
				return nil
			},
			ShutdownFunc: func() { calls++ },
		},
		SendMetrics: SendMetrics{
			BatchSendSizeFunc: sizeCall,
			InFlightFunc:      call,
			SentFunc:          call,
			SendFailureFunc:   func(context.Context, int64, ...metric.AddOption) { calls++ },
		},
	}

	ctx := context.Background()
	m.EnqueueFailure(ctx, 1)
	m.EnqueueSize(ctx, 1, nil)
	require.NoError(t, m.RegisterQueue(func() int64 { return 1 }, func() int64 { return 2 }))
	m.BatchSendSize(ctx, 1, nil)
	m.InFlight(ctx, 1)
	m.Sent(ctx, 1)
	m.SendFailure(ctx, 1)
	m.Shutdown()
	require.Equal(t, 8, calls)
}

func TestConfigWithObsMetrics(t *testing.T) {
	cfg := struct{}{}
	metrics := ObsMetrics{
		SendMetrics: SendMetrics{SentFunc: func(context.Context, int64) {}},
	}

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
