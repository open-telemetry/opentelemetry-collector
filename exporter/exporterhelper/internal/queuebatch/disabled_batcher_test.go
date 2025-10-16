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

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestDisabledBatcher(t *testing.T) {
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
			sink := requesttest.NewSink()
			ba := newDisabledBatcher(sink.Export)

			q, err := queue.NewQueue(queue.Settings[request.Request]{
				Capacity:        1000,
				BlockOnOverflow: true,
				NumConsumers:    tt.maxWorkers,
				Telemetry:       componenttest.NewNopTelemetrySettings(),
			}, ba.Consume)
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
