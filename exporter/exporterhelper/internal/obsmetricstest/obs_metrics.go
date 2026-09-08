// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package obsmetricstest provides test implementations of obsmetrics.ObsMetrics.
package obsmetricstest

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/obsmetrics"
)

// Nop implements obsmetrics.ObsMetrics without recording metrics.
type Nop struct{}

func (Nop) RecordEnqueueFailure(context.Context, int64)                   {}
func (Nop) RecordEnqueueSize(context.Context, int64, func() int64)        {}
func (Nop) RegisterQueueSize(func() int64) error                          { return nil }
func (Nop) RegisterQueueCapacity(func() int64) error                      { return nil }
func (Nop) RecordBatchSendSize(context.Context, int64, func() int64)      {}
func (Nop) RecordInFlight(context.Context, int64)                         {}
func (Nop) RecordSent(context.Context, int64)                             {}
func (Nop) RecordSendFailure(context.Context, int64, ...metric.AddOption) {}
func (Nop) Shutdown()                                                     {}

var _ obsmetrics.ObsMetrics = Nop{}
