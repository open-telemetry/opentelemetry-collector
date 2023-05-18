// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor"
)

const (
	scopeName = "go.opentelemetry.io/collector/processor/batchprocessor"
)

var (
	processorTagKey          = tag.MustNewKey(obsmetrics.ProcessorKey)
	statBatchSizeTriggerSend = stats.Int64("batch_size_trigger_send", "Number of times the batch was sent due to a size trigger", stats.UnitDimensionless)
	statTimeoutTriggerSend   = stats.Int64("timeout_trigger_send", "Number of times the batch was sent due to a timeout trigger", stats.UnitDimensionless)
	statBatchSendSize        = stats.Int64("batch_send_size", "Number of units in the batch", stats.UnitDimensionless)
	statBatchSendSizeBytes   = stats.Int64("batch_send_size_bytes", "Number of bytes in batch that was sent", stats.UnitBytes)
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
	processorTagKeys := []tag.Key{processorTagKey}

	countBatchSizeTriggerSendView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statBatchSizeTriggerSend.Name()),
		Measure:     statBatchSizeTriggerSend,
		Description: statBatchSizeTriggerSend.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	countTimeoutTriggerSendView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statTimeoutTriggerSend.Name()),
		Measure:     statTimeoutTriggerSend,
		Description: statTimeoutTriggerSend.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	distributionBatchSendSizeView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statBatchSendSize.Name()),
		Measure:     statBatchSendSize,
		Description: statBatchSendSize.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000),
	}

	distributionBatchSendSizeBytesView := &view.View{
		Name:        obsreport.BuildProcessorCustomMetricName(typeStr, statBatchSendSizeBytes.Name()),
		Measure:     statBatchSendSizeBytes,
		Description: statBatchSendSizeBytes.Description(),
		TagKeys:     processorTagKeys,
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

type batchProcessorTelemetry struct {
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

func newBatchProcessorTelemetry(set processor.CreateSettings, currentMetadataCardinality func() int, useOtel bool) (*batchProcessorTelemetry, error) {
	exportCtx, err := tag.New(context.Background(), tag.Insert(processorTagKey, set.ID.String()))
	if err != nil {
		return nil, err
	}

	bpt := &batchProcessorTelemetry{
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

func (bpt *batchProcessorTelemetry) createOtelMetrics(mp metric.MeterProvider, currentMetadataCardinality func() int) error {
	if !bpt.useOtel {
		return nil
	}

	var err error
	meter := mp.Meter(scopeName)

	bpt.batchSizeTriggerSend, err = meter.Int64Counter(
		obsreport.BuildProcessorCustomMetricName(typeStr, "batch_size_trigger_send"),
		metric.WithDescription("Number of times the batch was sent due to a size trigger"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	bpt.timeoutTriggerSend, err = meter.Int64Counter(
		obsreport.BuildProcessorCustomMetricName(typeStr, "timeout_trigger_send"),
		metric.WithDescription("Number of times the batch was sent due to a timeout trigger"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	bpt.batchSendSize, err = meter.Int64Histogram(
		obsreport.BuildProcessorCustomMetricName(typeStr, "batch_send_size"),
		metric.WithDescription("Number of units in the batch"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	bpt.batchSendSizeBytes, err = meter.Int64Histogram(
		obsreport.BuildProcessorCustomMetricName(typeStr, "batch_send_size_bytes"),
		metric.WithDescription("Number of bytes in batch that was sent"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	bpt.batchMetadataCardinality, err = meter.Int64ObservableUpDownCounter(
		obsreport.BuildProcessorCustomMetricName(typeStr, "metadata_cardinality"),
		metric.WithDescription("Number of distinct metadata value combinations being processed"),
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

func (bpt *batchProcessorTelemetry) record(trigger trigger, sent, bytes int64) {
	if bpt.useOtel {
		bpt.recordWithOtel(trigger, sent, bytes)
	} else {
		bpt.recordWithOC(trigger, sent, bytes)
	}
}

func (bpt *batchProcessorTelemetry) recordWithOC(trigger trigger, sent, bytes int64) {
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

func (bpt *batchProcessorTelemetry) recordWithOtel(trigger trigger, sent, bytes int64) {
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
