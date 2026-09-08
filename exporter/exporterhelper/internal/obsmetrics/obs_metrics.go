// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package obsmetrics defines the metrics reported by exporterhelper.
package obsmetrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
type ObsMetrics interface {
	RecordEnqueueFailure(ctx context.Context, items int64)
	RecordEnqueueSize(ctx context.Context, items int64, bytesSize func() int64)
	RegisterQueueSize(observeSize func() int64) error
	RegisterQueueCapacity(observeCapacity func() int64) error
	RecordBatchSendSize(ctx context.Context, items int64, bytesSize func() int64)
	RecordInFlight(ctx context.Context, delta int64)
	RecordSent(ctx context.Context, items int64)
	RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption)
	Shutdown()
}
