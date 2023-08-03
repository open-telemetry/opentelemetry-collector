// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var errNoValidSpanExporter = errors.New("no valid span exporter")

func newSpanProcessor(_ context.Context, processor SpanProcessor) (sdktrace.SpanProcessor, error) {
	if processor.Batch != nil {
		if processor.Batch.Exporter.Console != nil {
			exp, err := stdouttrace.New(
				stdouttrace.WithPrettyPrint(),
			)
			if err != nil {
				return nil, err
			}
			opts := []sdktrace.BatchSpanProcessorOption{}
			if processor.Batch.ExportTimeout != nil {
				if *processor.Batch.ExportTimeout < 0 {
					return nil, fmt.Errorf("invalid export timeout %d", *processor.Batch.ExportTimeout)
				}
				opts = append(opts, sdktrace.WithExportTimeout(time.Millisecond*time.Duration(*processor.Batch.ExportTimeout)))
			}
			if processor.Batch.MaxExportBatchSize != nil {
				if *processor.Batch.MaxExportBatchSize < 0 {
					return nil, fmt.Errorf("invalid batch size %d", *processor.Batch.MaxExportBatchSize)
				}
				opts = append(opts, sdktrace.WithMaxExportBatchSize(*processor.Batch.MaxExportBatchSize))
			}
			if processor.Batch.MaxQueueSize != nil {
				if *processor.Batch.MaxQueueSize < 0 {
					return nil, fmt.Errorf("invalid queue size %d", *processor.Batch.MaxQueueSize)
				}
				opts = append(opts, sdktrace.WithMaxQueueSize(*processor.Batch.MaxQueueSize))
			}
			if processor.Batch.ScheduleDelay != nil {
				if *processor.Batch.ScheduleDelay < 0 {
					return nil, fmt.Errorf("invalid schedule delay %d", *processor.Batch.ScheduleDelay)
				}
				opts = append(opts, sdktrace.WithBatchTimeout(time.Millisecond*time.Duration(*processor.Batch.ScheduleDelay)))
			}
			return sdktrace.NewBatchSpanProcessor(exp, opts...), nil
		}
		return nil, errNoValidSpanExporter
	}
	return nil, fmt.Errorf("unsupported span processor type %v", processor)
}
