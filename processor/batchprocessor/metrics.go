// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/internal"
)

type trigger int

const (
	triggerTimeout trigger = iota
	triggerBatchSize
)

type batchProcessorTelemetry struct {
	detailed bool

	exportCtx context.Context

	processorAttr    metric.MeasurementOption
	telemetryBuilder *metadata.TelemetryBuilder
}

func newBatchProcessorTelemetry(set processor.Settings, currentMetadataCardinality func() int) (*batchProcessorTelemetry, error) {
	attrs := metric.WithAttributeSet(attribute.NewSet(attribute.String(internal.ProcessorKey, set.ID.String())))

	telemetryBuilder, err := metadata.NewTelemetryBuilder(
		set.TelemetrySettings,
		metadata.WithProcessorBatchMetadataCardinalityCallback(func() int64 { return int64(currentMetadataCardinality()) }, attrs),
	)

	if err != nil {
		return nil, err
	}

	return &batchProcessorTelemetry{
		exportCtx:        context.Background(),
		detailed:         set.MetricsLevel == configtelemetry.LevelDetailed,
		telemetryBuilder: telemetryBuilder,
		processorAttr:    attrs,
	}, nil
}

func (bpt *batchProcessorTelemetry) record(trigger trigger, sent, bytes int64) {
	switch trigger {
	case triggerBatchSize:
		bpt.telemetryBuilder.ProcessorBatchBatchSizeTriggerSend.Add(bpt.exportCtx, 1, bpt.processorAttr)
	case triggerTimeout:
		bpt.telemetryBuilder.ProcessorBatchTimeoutTriggerSend.Add(bpt.exportCtx, 1, bpt.processorAttr)
	}

	bpt.telemetryBuilder.ProcessorBatchBatchSendSize.Record(bpt.exportCtx, sent, bpt.processorAttr)
	bpt.telemetryBuilder.ProcessorBatchBatchSendSizeBytes.Record(bpt.exportCtx, bytes, bpt.processorAttr)
}
