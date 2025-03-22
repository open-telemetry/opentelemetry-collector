// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
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

			sink := requesttest.NewSink()
			ba, err := NewBatcher(cfg, sink.Export, tt.maxWorkers)
			require.NoError(t, err)

			q, err := NewQueue[request.Request](
				context.Background(),
				QueueSettings[request.Request]{
					Signal:    pipeline.SignalTraces,
					ID:        component.NewID(exportertest.NopType),
					Telemetry: componenttest.NewNopTelemetrySettings(),
					Settings: Settings[request.Request]{
						Sizers: map[exporterbatcher.SizerType]Sizer[request.Request]{
							exporterbatcher.SizerTypeRequests: RequestsSizer[request.Request]{},
						},
					},
				},
				exporterqueue.NewDefaultConfig(),
				ba.Consume)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8}))
			sink.SetExportErr(errors.New("transient error"))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 8}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 17}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 13}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 35}))
			require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: 2}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && sink.ItemsCount() == 75
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
}
