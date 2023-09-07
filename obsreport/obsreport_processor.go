// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"context"
	"strings"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/processor"
)

var (
	processorName  = "processor"
	processorScope = scopeName + nameSep + processorName
)

// BuildProcessorCustomMetricName is used to be build a metric name following
// the standards used in the Collector. The configType should be the same
// value used to identify the type on the config.
func BuildProcessorCustomMetricName(configType, metric string) string {
	componentPrefix := obsmetrics.ProcessorPrefix
	if !strings.HasSuffix(componentPrefix, obsmetrics.NameSep) {
		componentPrefix += obsmetrics.NameSep
	}
	if configType == "" {
		return componentPrefix
	}
	return componentPrefix + configType + obsmetrics.NameSep + metric
}

// Processor is a helper to add observability to a processor.
type Processor struct {
	level    configtelemetry.Level
	mutators []tag.Mutator

	logger *zap.Logger

	useOtelForMetrics bool
	otelAttrs         []attribute.KeyValue

	acceptedSpansCounter        metric.Int64Counter
	refusedSpansCounter         metric.Int64Counter
	droppedSpansCounter         metric.Int64Counter
	acceptedMetricPointsCounter metric.Int64Counter
	refusedMetricPointsCounter  metric.Int64Counter
	droppedMetricPointsCounter  metric.Int64Counter
	acceptedLogRecordsCounter   metric.Int64Counter
	refusedLogRecordsCounter    metric.Int64Counter
	droppedLogRecordsCounter    metric.Int64Counter
}

// ProcessorSettings are settings for creating a Processor.
type ProcessorSettings struct {
	ProcessorID             component.ID
	ProcessorCreateSettings processor.CreateSettings
}

// NewProcessor creates a new Processor.
func NewProcessor(cfg ProcessorSettings) (*Processor, error) {
	return newProcessor(cfg, obsreportconfig.UseOtelForInternalMetricsfeatureGate.IsEnabled())
}

func newProcessor(cfg ProcessorSettings, useOtel bool) (*Processor, error) {
	proc := &Processor{
		level:             cfg.ProcessorCreateSettings.MetricsLevel,
		mutators:          []tag.Mutator{tag.Upsert(obsmetrics.TagKeyProcessor, cfg.ProcessorID.String(), tag.WithTTL(tag.TTLNoPropagation))},
		logger:            cfg.ProcessorCreateSettings.Logger,
		useOtelForMetrics: useOtel,
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ProcessorKey, cfg.ProcessorID.String()),
		},
	}

	// ignore returned error as per workaround in https://github.com/open-telemetry/opentelemetry-collector/issues/8346
	// if err := proc.createOtelMetrics(cfg); err != nil {
	// 	return nil, err
	// }
	_ = proc.createOtelMetrics(cfg)

	return proc, nil
}

func (por *Processor) createOtelMetrics(cfg ProcessorSettings) error {
	if !por.useOtelForMetrics {
		return nil
	}
	meter := cfg.ProcessorCreateSettings.MeterProvider.Meter(processorScope)
	var errors, err error

	por.acceptedSpansCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.AcceptedSpansKey,
		metric.WithDescription("Number of spans successfully pushed into the next component in the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.refusedSpansCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.RefusedSpansKey,
		metric.WithDescription("Number of spans that were rejected by the next component in the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.droppedSpansCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.DroppedSpansKey,
		metric.WithDescription("Number of spans that were dropped."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.acceptedMetricPointsCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.AcceptedMetricPointsKey,
		metric.WithDescription("Number of metric points successfully pushed into the next component in the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.refusedMetricPointsCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.RefusedMetricPointsKey,
		metric.WithDescription("Number of metric points that were rejected by the next component in the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.droppedMetricPointsCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.DroppedMetricPointsKey,
		metric.WithDescription("Number of metric points that were dropped."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.acceptedLogRecordsCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.AcceptedLogRecordsKey,
		metric.WithDescription("Number of log records successfully pushed into the next component in the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.refusedLogRecordsCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.RefusedLogRecordsKey,
		metric.WithDescription("Number of log records that were rejected by the next component in the pipeline."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	por.droppedLogRecordsCounter, err = meter.Int64Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.DroppedLogRecordsKey,
		metric.WithDescription("Number of log records that were dropped."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	return errors
}

func (por *Processor) recordWithOtel(ctx context.Context, dataType component.DataType, accepted, refused, dropped int64) {
	var acceptedCount, refusedCount, droppedCount metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		acceptedCount = por.acceptedSpansCounter
		refusedCount = por.refusedSpansCounter
		droppedCount = por.droppedSpansCounter
	case component.DataTypeMetrics:
		acceptedCount = por.acceptedMetricPointsCounter
		refusedCount = por.refusedMetricPointsCounter
		droppedCount = por.droppedMetricPointsCounter
	case component.DataTypeLogs:
		acceptedCount = por.acceptedLogRecordsCounter
		refusedCount = por.refusedLogRecordsCounter
		droppedCount = por.droppedLogRecordsCounter
	}

	acceptedCount.Add(ctx, accepted, metric.WithAttributes(por.otelAttrs...))
	refusedCount.Add(ctx, refused, metric.WithAttributes(por.otelAttrs...))
	droppedCount.Add(ctx, dropped, metric.WithAttributes(por.otelAttrs...))
}

func (por *Processor) recordWithOC(ctx context.Context, dataType component.DataType, accepted, refused, dropped int64) {
	var acceptedMeasure, refusedMeasure, droppedMeasure *stats.Int64Measure

	switch dataType {
	case component.DataTypeTraces:
		acceptedMeasure = obsmetrics.ProcessorAcceptedSpans
		refusedMeasure = obsmetrics.ProcessorRefusedSpans
		droppedMeasure = obsmetrics.ProcessorDroppedSpans
	case component.DataTypeMetrics:
		acceptedMeasure = obsmetrics.ProcessorAcceptedMetricPoints
		refusedMeasure = obsmetrics.ProcessorRefusedMetricPoints
		droppedMeasure = obsmetrics.ProcessorDroppedMetricPoints
	case component.DataTypeLogs:
		acceptedMeasure = obsmetrics.ProcessorAcceptedLogRecords
		refusedMeasure = obsmetrics.ProcessorRefusedLogRecords
		droppedMeasure = obsmetrics.ProcessorDroppedLogRecords
	}

	// ignore the error for now; should not happen
	_ = stats.RecordWithTags(
		ctx,
		por.mutators,
		acceptedMeasure.M(accepted),
		refusedMeasure.M(refused),
		droppedMeasure.M(dropped),
	)
}

func (por *Processor) recordData(ctx context.Context, dataType component.DataType, accepted, refused, dropped int64) {
	if por.useOtelForMetrics {
		por.recordWithOtel(ctx, dataType, accepted, refused, dropped)
	} else {
		por.recordWithOC(ctx, dataType, accepted, refused, dropped)
	}
}

// TracesAccepted reports that the trace data was accepted.
func (por *Processor) TracesAccepted(ctx context.Context, numSpans int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeTraces, int64(numSpans), int64(0), int64(0))
	}
}

// TracesRefused reports that the trace data was refused.
func (por *Processor) TracesRefused(ctx context.Context, numSpans int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeTraces, int64(0), int64(numSpans), int64(0))
	}
}

// TracesDropped reports that the trace data was dropped.
func (por *Processor) TracesDropped(ctx context.Context, numSpans int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeTraces, int64(0), int64(0), int64(numSpans))
	}
}

// MetricsAccepted reports that the metrics were accepted.
func (por *Processor) MetricsAccepted(ctx context.Context, numPoints int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeMetrics, int64(numPoints), int64(0), int64(0))
	}
}

// MetricsRefused reports that the metrics were refused.
func (por *Processor) MetricsRefused(ctx context.Context, numPoints int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeMetrics, int64(0), int64(numPoints), int64(0))
	}
}

// MetricsDropped reports that the metrics were dropped.
func (por *Processor) MetricsDropped(ctx context.Context, numPoints int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeMetrics, int64(0), int64(0), int64(numPoints))
	}
}

// LogsAccepted reports that the logs were accepted.
func (por *Processor) LogsAccepted(ctx context.Context, numRecords int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeLogs, int64(numRecords), int64(0), int64(0))
	}
}

// LogsRefused reports that the logs were refused.
func (por *Processor) LogsRefused(ctx context.Context, numRecords int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeLogs, int64(0), int64(numRecords), int64(0))
	}
}

// LogsDropped reports that the logs were dropped.
func (por *Processor) LogsDropped(ctx context.Context, numRecords int) {
	if por.level != configtelemetry.LevelNone {
		por.recordData(ctx, component.DataTypeLogs, int64(0), int64(0), int64(numRecords))
	}
}
