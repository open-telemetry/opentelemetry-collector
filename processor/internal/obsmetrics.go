// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/processor/internal"

const (
	MetricNameSep = "_"

	ProcessorKey = "processor"
	SignalKey    = "otel.signal"
	PipelineKey  = "pipeline"

	ProcessorMetricPrefix = ProcessorKey + MetricNameSep
)
