// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestDisabledBatcher_Basic(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{
			name:       "one_worker",
			maxWorkers: 1,
		},
		{
			name:       "three_workers",
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = false

			ba, err := NewBatcher(cfg,
				func(ctx context.Context, req request.Request) error { return req.Export(ctx) },
				tt.maxWorkers)
			require.NoError(t, err)

			q := exporterqueue.NewQueue[request.Request](
				context.Background(),
				exporterqueue.Settings[request.Request]{
					Signal:           pipeline.SignalTraces,
					ExporterSettings: exportertest.NewNopSettings(exportertest.NopType),
				},
				exporterqueue.NewDefaultConfig(),
				ba.Consume)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8, ExportErr: errors.New("transient error"), Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 17, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35, Sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && sink.ItemsCount() == 75
			}, 30*time.Millisecond, 10*time.Millisecond)
		})
	}
}
