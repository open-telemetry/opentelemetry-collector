// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
)

type trigger int

const (
	typeStr                = "batch"
	triggerTimeout trigger = iota
	triggerBatchSize
)

type batchProcessorTelemetry struct {
	detailed bool

	exportCtx context.Context

	processorAttr    attribute.Set
	telemetryBuilder *metadata.TelemetryBuilder
}

func newBatchProcessorTelemetry(set processor.CreateSettings, currentMetadataCardinality func() int) (*batchProcessorTelemetry, error) {
	attrs := attribute.NewSet(attribute.String(obsmetrics.ProcessorKey, set.ID.String()))

	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings,
		metadata.WithLevel(set.MetricsLevel),
		metadata.WithProcessorBatchMetadataCardinalityCallback(func() int64 { return int64(currentMetadataCardinality()) }),
		metadata.WithAttributeSet(attrs),
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
		bpt.telemetryBuilder.ProcessorBatchBatchSizeTriggerSend.Add(bpt.exportCtx, 1, metric.WithAttributeSet(bpt.processorAttr))
	case triggerTimeout:
		bpt.telemetryBuilder.ProcessorBatchTimeoutTriggerSend.Add(bpt.exportCtx, 1, metric.WithAttributeSet(bpt.processorAttr))
	}

	bpt.telemetryBuilder.ProcessorBatchBatchSendSize.Record(bpt.exportCtx, sent, metric.WithAttributeSet(bpt.processorAttr))
	if bpt.detailed {
		bpt.telemetryBuilder.ProcessorBatchBatchSendSizeBytes.Record(bpt.exportCtx, bytes, metric.WithAttributeSet(bpt.processorAttr))
	}
}
