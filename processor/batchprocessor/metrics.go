// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type trigger int

const (
	typeStr                = "batch"
	triggerTimeout trigger = iota
	triggerBatchSize
)

type batchProcessorTelemetry struct {
	level    configtelemetry.Level
	detailed bool

	exportCtx context.Context

	processorAttr            []attribute.KeyValue
	batchSizeTriggerSend     metric.Int64Counter
	timeoutTriggerSend       metric.Int64Counter
	batchSendSize            metric.Int64Histogram
	batchSendSizeBytes       metric.Int64Histogram
	batchMetadataCardinality metric.Int64ObservableUpDownCounter
}

func newBatchProcessorTelemetry(set processor.CreateSettings, currentMetadataCardinality func() int) (*batchProcessorTelemetry, error) {
	bpt := &batchProcessorTelemetry{
		processorAttr: []attribute.KeyValue{attribute.String(obsmetrics.ProcessorKey, set.ID.String())},
		exportCtx:     context.Background(),
		level:         set.MetricsLevel,
		detailed:      set.MetricsLevel == configtelemetry.LevelDetailed,
	}

	if err := bpt.createOtelMetrics(set.TelemetrySettings, currentMetadataCardinality); err != nil {
		return nil, err
	}

	return bpt, nil
}

func (bpt *batchProcessorTelemetry) createOtelMetrics(set component.TelemetrySettings, currentMetadataCardinality func() int) error {
	var (
		errors, err error
		meter       metric.Meter
	)

	// BatchProcessor are emitted starting from Normal level only.
	if bpt.level >= configtelemetry.LevelNormal {
		meter = metadata.Meter(set)
	} else {
		meter = noopmetric.Meter{}
	}

	bpt.batchSizeTriggerSend, err = meter.Int64Counter(
		processorhelper.BuildCustomMetricName(typeStr, "batch_size_trigger_send"),
		metric.WithDescription("Number of times the batch was sent due to a size trigger"),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	bpt.timeoutTriggerSend, err = meter.Int64Counter(
		processorhelper.BuildCustomMetricName(typeStr, "timeout_trigger_send"),
		metric.WithDescription("Number of times the batch was sent due to a timeout trigger"),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchSendSize, err = meter.Int64Histogram(
		processorhelper.BuildCustomMetricName(typeStr, "batch_send_size"),
		metric.WithDescription("Number of units in the batch"),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchSendSizeBytes, err = meter.Int64Histogram(
		processorhelper.BuildCustomMetricName(typeStr, "batch_send_size_bytes"),
		metric.WithDescription("Number of bytes in batch that was sent"),
		metric.WithUnit("By"),
	)
	errors = multierr.Append(errors, err)

	bpt.batchMetadataCardinality, err = meter.Int64ObservableUpDownCounter(
		processorhelper.BuildCustomMetricName(typeStr, "metadata_cardinality"),
		metric.WithDescription("Number of distinct metadata value combinations being processed"),
		metric.WithUnit("1"),
		metric.WithInt64Callback(func(_ context.Context, obs metric.Int64Observer) error {
			obs.Observe(int64(currentMetadataCardinality()))
			return nil
		}),
	)
	errors = multierr.Append(errors, err)

	return errors
}

func (bpt *batchProcessorTelemetry) record(trigger trigger, sent, bytes int64) {
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
