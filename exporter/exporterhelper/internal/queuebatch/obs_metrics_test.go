// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

func TestQueueBatchMetrics(t *testing.T) {
	var gotItems, gotBytes int64
	metrics := NewQueueBatchMetrics(
		WithQueueMetrics(nil),
		WithRecordBatchSendSize(func(_ context.Context, items int64, bytesSize queue.Int64Value) {
			gotItems = items
			gotBytes = bytesSize.Value()
		}),
	)

	metrics.RecordBatchSendSize(context.Background(), 3, queue.NewInt64Value(func() int64 { return 7 }))
	require.Equal(t, int64(3), gotItems)
	require.Equal(t, int64(7), gotBytes)
}

func TestQueueBatchMetricsWithoutRecorder(_ *testing.T) {
	NewQueueBatchMetrics().RecordBatchSendSize(context.Background(), 1, nil)
}
