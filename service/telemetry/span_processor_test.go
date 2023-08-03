// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func intPtr(i int) *int {
	return &i
}

func TestSpanProcessor(t *testing.T) {
	testCases := []struct {
		name      string
		processor SpanProcessor
		args      any
		err       error
	}{
		{
			name: "no processor",
			err:  errors.New("unsupported span processor type {<nil> <nil>}"),
		},
		{
			name: "batch processor invalid exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			err: errNoValidSpanExporter,
		},
		{
			name: "batch processor invalid batch size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(-1),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			err: errors.New("invalid batch size -1"),
		},
		{
			name: "batch processor invalid export timeout console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					ExportTimeout: intPtr(-2),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			err: errors.New("invalid export timeout -2"),
		},
		{
			name: "batch processor invalid queue size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxQueueSize: intPtr(-3),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			err: errors.New("invalid queue size -3"),
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					ScheduleDelay: intPtr(-4),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			err: errors.New("invalid schedule delay -4"),
		},
		{
			name: "batch processor console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSpanProcessor(context.Background(), tt.processor)
			assert.Equal(t, tt.err, err)
		})
	}
}
