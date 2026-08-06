// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
// Exporterhelper owns the meaning and timing of each operation; the
// implementation supplies the instruments. Components that reuse
// exporterhelper implement this to report metrics using names and attributes
// appropriate for that component, most easily by setting the operations of a
// FuncObsMetrics.
type ObsMetrics = internal.ObsMetrics

// FuncObsMetrics implements ObsMetrics from a set of operations. The zero
// value reports nothing, so a component only sets the events it reports.
type FuncObsMetrics = internal.FuncObsMetrics

// RecordEnqueueFailureFunc records the number of items dropped because they
// could not be added to the queue.
type RecordEnqueueFailureFunc = queue.RecordEnqueueFailureFunc

// RecordEnqueueItemsFunc records the number of items and bytes in a request
// as it is offered to the queue.
type RecordEnqueueItemsFunc = queue.RecordEnqueueItemsFunc

// RegisterQueueSizeFunc installs an observer for the current queue size.
type RegisterQueueSizeFunc = queue.RegisterQueueSizeFunc

// RegisterQueueCapacityFunc installs an observer for the fixed queue capacity.
type RegisterQueueCapacityFunc = queue.RegisterQueueCapacityFunc

// RecordInFlightFunc records a change in the number of requests currently
// being sent. Delta is +1 when a send starts and -1 when it ends.
type RecordInFlightFunc = internal.RecordInFlightFunc

// RecordSentFunc records the number of items successfully sent.
type RecordSentFunc = internal.RecordSentFunc

// RecordSendFailureFunc records the number of items that failed to send. The
// options carry the failure attributes derived from the error.
type RecordSendFailureFunc = internal.RecordSendFailureFunc

// ShutdownObsMetricsFunc releases the resources backing the instruments.
type ShutdownObsMetricsFunc = internal.ShutdownObsMetricsFunc
