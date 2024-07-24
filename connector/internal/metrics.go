// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/connector/internal"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// A Metrics connector acts as an exporter from a metrics pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Metrics feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Latency between related data points could be modeled and emitted as traces.
//   - Metrics could be collected in one pipeline and routed to another metrics pipeline
//     based on criteria such as attributes or other content of the metric. The second
//     pipeline can then process and export the metric to the appropriate backend.
//   - Metrics could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Metrics interface {
	component.Component
	consumer.Metrics
}
