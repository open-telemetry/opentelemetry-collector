// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"context"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	scopeName = "go.opentelemetry.io/collector/exporter/exporterhelper/exporterbatcher"
)

var (
	exporterTagKey           = tag.MustNewKey(obsmetrics.ExporterKey)
	statBatchSizeTriggerSend = stats.Int64("batch_size_trigger_send", "Number of times the batch was sent due to a size trigger", stats.UnitDimensionless)
	statTimeoutTriggerSend   = stats.Int64("batch_timeout_trigger_send",
		"Number of times the batch was sent due to a timeout trigger", stats.UnitDimensionless)
	statBatchSendSize      = stats.Int64("batch_send_size", "Number of units in the batch", stats.UnitDimensionless)
	statBatchSendSizeBytes = stats.Int64("batch_send_size_bytes", "Number of bytes in batch that was sent", stats.UnitBytes)
)

type trigger int

const (
	triggerTimeout trigger = iota
	triggerBatchSize
)

func init() {
	// TODO: Find a way to handle the error.
	_ = view.Register(metricViews()...)
}

// MetricViews returns the metrics views related to batching
func metricViews() []*view.View {
	exporterTagKeys := []tag.Key{exporterTagKey}

	countBatchSizeTriggerSendView := &view.View{
		Name:        obsmetrics.ExporterPrefix + statBatchSizeTriggerSend.Name(),
		Measure:     statBatchSizeTriggerSend,
		Description: statBatchSizeTriggerSend.Description(),
		TagKeys:     exporterTagKeys,
		Aggregation: view.Sum(),
	}

	countTimeoutTriggerSendView := &view.View{
		Name:        obsmetrics.ExporterPrefix + obsmetrics.NameSep + statTimeoutTriggerSend.Name(),
		Measure:     statTimeoutTriggerSend,
		Description: statTimeoutTriggerSend.Description(),
		TagKeys:     exporterTagKeys,
		Aggregation: view.Sum(),
	}

	distributionBatchSendSizeView := &view.View{
		Name:        obsmetrics.ExporterPrefix + obsmetrics.NameSep + statBatchSendSize.Name(),
		Measure:     statBatchSendSize,
		Description: statBatchSendSize.Description(),
		TagKeys:     exporterTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000),
	}

	distributionBatchSendSizeBytesView := &view.View{
		Name:        obsmetrics.ExporterPrefix + obsmetrics.NameSep + statBatchSendSizeBytes.Name(),
		Measure:     statBatchSendSizeBytes,
		Description: statBatchSendSizeBytes.Description(),
		TagKeys:     exporterTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
			100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
			1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000),
	}

	return []*view.View{
		countBatchSizeTriggerSendView,
		countTimeoutTriggerSendView,
		distributionBatchSendSizeView,
		distributionBatchSendSizeBytesView,
	}
}

type batchTelemetry struct {
	level    configtelemetry.Level
	detailed bool
	useOtel  bool

	exportCtx context.Context

	processorAttr            []attribute.KeyValue
	batchSizeTriggerSend     metric.Int64Counter
	timeoutTriggerSend       metric.Int64Counter
	batchSendSize            metric.Int64Histogram
	batchSendSizeBytes       metric.Int64Histogram
	batchMetadataCardinality metric.Int64ObservableUpDownCounter
}

func newBatchTelemetry(set exporter.CreateSettings, currentMetadataCardinality func() int,
	useOtel bool) (*batchTelemetry, error) {
	exportCtx, err := tag.New(context.Background(), tag.Insert(exporterTagKey, set.ID.String()))
	if err != nil {
		return nil, err
	}

	bpt := &batchTelemetry{
		useOtel:       useOtel,
		processorAttr: []attribute.KeyValue{attribute.String(obsmetrics.ProcessorKey, set.ID.String())},
		exportCtx:     exportCtx,
		level:         set.MetricsLevel,
		detailed:      set.MetricsLevel == configtelemetry.LevelDetailed,
	}

	err = bpt.createOtelMetrics(set.MeterProvider, currentMetadataCardinality)
	if err != nil {
		return nil, err
	}

	return bpt, nil
}

func (bpt *batchTelemetry) createOtelMetrics(mp metric.MeterProvider,
	currentMetadataCardinality func() int) error {
	if !bpt.useOtel {
		return nil
	}

	var err error
	meter := mp.Meter(scopeName)

	bpt.batchSizeTriggerSend, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.NameSep+"batch_size_trigger_send",
		metric.WithDescription("Number of times the batch was sent due to a size trigger"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	bpt.timeoutTriggerSend, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.NameSep+"timeout_trigger_send",
		metric.WithDescription("Number of times the batch was sent due to a timeout trigger"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	bpt.batchSendSize, err = meter.Int64Histogram(
		obsmetrics.ExporterPrefix+obsmetrics.NameSep+"batch_send_size",
		metric.WithDescription("Number of units in the batch"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	bpt.batchSendSizeBytes, err = meter.Int64Histogram(
		obsmetrics.ExporterPrefix+obsmetrics.NameSep+"batch_send_size_bytes",
		metric.WithDescription("Number of bytes in batch that was sent"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	bpt.batchMetadataCardinality, err = meter.Int64ObservableUpDownCounter(
		obsmetrics.ExporterPrefix+obsmetrics.NameSep+"batch_identifies_cardinality",
		metric.WithDescription("Number of distinct batch identifies value combinations being processed"),
		metric.WithUnit("1"),
		metric.WithInt64Callback(func(_ context.Context, obs metric.Int64Observer) error {
			obs.Observe(int64(currentMetadataCardinality()))
			return nil
		}),
	)
	if err != nil {
		return err
	}

	return nil
}

func (bpt *batchTelemetry) record(trigger trigger, sent, bytes int64) {
	if bpt.useOtel {
		bpt.recordWithOtel(trigger, sent, bytes)
	} else {
		bpt.recordWithOC(trigger, sent, bytes)
	}
}

func (bpt *batchTelemetry) recordWithOC(trigger trigger, sent, bytes int64) {
	var triggerMeasure *stats.Int64Measure
	switch trigger {
	case triggerBatchSize:
		triggerMeasure = statBatchSizeTriggerSend
	case triggerTimeout:
		triggerMeasure = statTimeoutTriggerSend
	}

	stats.Record(bpt.exportCtx, triggerMeasure.M(1), statBatchSendSize.M(sent))
	if bpt.detailed {
		stats.Record(bpt.exportCtx, statBatchSendSizeBytes.M(bytes))
	}
}

func (bpt *batchTelemetry) recordWithOtel(trigger trigger, sent, bytes int64) {
	switch trigger {
	case triggerBatchSize:
		bpt.batchSizeTriggerSend.Add(bpt.exportCtx, 1, metric.WithAttributes(bpt.processorAttr...))
	case triggerTimeout:
		bpt.timeoutTriggerSend.Add(bpt.exportCtx, 1, metric.WithAttributes(bpt.processorAttr...))
	}

	bpt.batchSendSize.Record(bpt.exportCtx, sent, metric.WithAttributes(bpt.processorAttr...))
	if bpt.detailed {
		bpt.batchSendSizeBytes.Record(bpt.exportCtx, bytes, metric.WithAttributes(bpt.processorAttr...))
	}
}
