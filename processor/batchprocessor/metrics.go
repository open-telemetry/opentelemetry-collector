// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	scopeName = "github.com/open-telemetry/otel-arrow/collector/processor/concurrentbatchprocessor"
)

var (
	processorTagKey = ctxKey("processor")
)

type trigger int
type ctxKey string

const (
	triggerTimeout trigger = iota
	triggerBatchSize
	triggerShutdown

	// metricTypeStr is the name used in metrics, so that this component can be
	// monitored using the same metric names of the upstream batchprocessor.
	// They still have different `processor` attributes.
	metricTypeStr = "batch"
)

type batchProcessorTelemetry struct {
	level    configtelemetry.Level
	detailed bool

	exportCtx context.Context

	processorAttr            []attribute.KeyValue
	processorAttrOption      metric.MeasurementOption
	batchSizeTriggerSend     metric.Int64Counter
	timeoutTriggerSend       metric.Int64Counter
	batchSendSize            metric.Int64Histogram
	batchSendSizeBytes       metric.Int64Histogram
	batchSendLatency         metric.Float64Histogram
	batchMetadataCardinality metric.Int64ObservableUpDownCounter

	// Note: since the semaphore does not provide access to its current
	// value, we instrument the number of in-flight bytes using parallel
	// instrumentation counting acquired and released bytes.
	batchInFlightBytes metric.Int64UpDownCounter
}

func newBatchProcessorTelemetry(set processor.Settings, currentMetadataCardinality func() int) (*batchProcessorTelemetry, error) {
	exportCtx := context.WithValue(context.Background(), processorTagKey, set.ID.String())

	bpt := &batchProcessorTelemetry{
		processorAttr: []attribute.KeyValue{attribute.String("processor", set.ID.String())},
		exportCtx:     exportCtx,
		level:         set.MetricsLevel,
		detailed:      set.MetricsLevel == configtelemetry.LevelDetailed,
	}

	if err := bpt.createOtelMetrics(set.MeterProvider, currentMetadataCardinality); err != nil {
		return nil, err
	}

	return bpt, nil
}

func (bpt *batchProcessorTelemetry) createOtelMetrics(mp metric.MeterProvider, currentMetadataCardinality func() int) error {
	bpt.processorAttrOption = metric.WithAttributes(bpt.processorAttr...)

	var errors, err error
	meter := mp.Meter(scopeName)

	bpt.batchSizeTriggerSend, err = meter.Int64Counter(
		processorhelper.BuildCustomMetricName(metricTypeStr, "batch_size_trigger_send"),
		metric.WithDescription("Number of times the batch was sent due to a size trigger"),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	bpt.timeoutTriggerSend, err = meter.Int64Counter(
		processorhelper.BuildCustomMetricName(metricTypeStr, "timeout_trigger_send"),
		metric.WithDescription("Number of times the batch was sent due to a timeout trigger"),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchSendSize, err = meter.Int64Histogram(
		processorhelper.BuildCustomMetricName(metricTypeStr, "batch_send_size"),
		metric.WithDescription("Number of units in the batch"),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchSendSizeBytes, err = meter.Int64Histogram(
		processorhelper.BuildCustomMetricName(metricTypeStr, "batch_send_size_bytes"),
		metric.WithDescription("Number of bytes in batch that was sent"),
		metric.WithUnit("By"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchSendLatency, err = meter.Float64Histogram(
		processorhelper.BuildCustomMetricName(metricTypeStr, "batch_send_latency"),
		metric.WithDescription("Duration of the export request"),
		metric.WithUnit("s"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchMetadataCardinality, err = meter.Int64ObservableUpDownCounter(
		processorhelper.BuildCustomMetricName(metricTypeStr, "metadata_cardinality"),
		metric.WithDescription("Number of distinct metadata value combinations being processed"),
		metric.WithUnit("1"),
		metric.WithInt64Callback(func(_ context.Context, obs metric.Int64Observer) error {
			obs.Observe(int64(currentMetadataCardinality()), bpt.processorAttrOption)
			return nil
		}),
	)
	errors = multierr.Append(errors, err)

	bpt.batchInFlightBytes, err = meter.Int64UpDownCounter(
		processorhelper.BuildCustomMetricName(metricTypeStr, "in_flight_bytes"),
		metric.WithDescription("Number of bytes in flight"),
		metric.WithUnit("By"),
	)
	errors = multierr.Append(errors, err)

	return errors
}

func (bpt *batchProcessorTelemetry) record(latency time.Duration, trigger trigger, sent, bytes int64) {
	bpt.recordWithOtel(latency, trigger, sent, bytes)
}

func (bpt *batchProcessorTelemetry) recordWithOtel(latency time.Duration, trigger trigger, sent, bytes int64) {
	switch trigger {
	case triggerBatchSize:
		bpt.batchSizeTriggerSend.Add(bpt.exportCtx, 1, bpt.processorAttrOption)
	case triggerTimeout:
		bpt.timeoutTriggerSend.Add(bpt.exportCtx, 1, bpt.processorAttrOption)
	}

	bpt.batchSendSize.Record(bpt.exportCtx, sent, bpt.processorAttrOption)
	if bpt.detailed {
		bpt.batchSendLatency.Record(bpt.exportCtx, latency.Seconds(), bpt.processorAttrOption)
		bpt.batchSendSizeBytes.Record(bpt.exportCtx, bytes, bpt.processorAttrOption)
	}
}
