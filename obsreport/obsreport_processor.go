// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// BuildProcessorCustomMetricName is used to be build a metric name following
// the standards used in the Collector. The configType should be the same
// value used to identify the type on the config.
//
// Deprecated: [0.86.0] Use processorhelper.BuildCustomMetricName instead.
func BuildProcessorCustomMetricName(configType, metric string) string {
	return processorhelper.BuildCustomMetricName(configType, metric)
}

// Processor is a helper to add observability to a processor.
//
// Deprecated: [0.86.0] Use processorhelper.ObsReport instead.
type Processor = processorhelper.ObsReport

// ProcessorSettings is a helper to add observability to a processor.
//
// Deprecated: [0.86.0] Use processorhelper.ObsReportSettings instead.
type ProcessorSettings = processorhelper.ObsReportSettings

// NewProcessor creates a new Processor.
//
// Deprecated: [0.86.0] Use processorhelper.NewObsReport instead.
func NewProcessor(cfg ProcessorSettings) (*Processor, error) {
	return processorhelper.NewObsReport(cfg)
}
