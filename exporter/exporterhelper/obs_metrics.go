// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

// QueueObserver returns the current size or capacity of a queue.
// Exporterhelper passes an observer to the RegisterQueueSize and
// RegisterQueueCapacity operations, which are expected to install it as an
// asynchronous callback.
type QueueObserver = queue.QueueObserver

// ObsMetricsConfig defines the operations invoked by exporterhelper to report
// its observation events. Exporterhelper owns the meaning and timing of each
// operation; a component supplies the instruments. Nil operations are no-ops,
// so a component only implements the events it reports.
type ObsMetricsConfig = internal.ObsMetricsConfig

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
type ObsMetrics = internal.ObsMetrics

// NewObsMetrics creates ObsMetrics that report through the operations in cfg.
func NewObsMetrics(cfg ObsMetricsConfig) *ObsMetrics {
	return internal.NewObsMetrics(cfg)
}
