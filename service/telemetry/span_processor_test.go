// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

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
		{
			name: "batch/otlp-exporter-invalid-protocol",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol: *strPtr("http/invalid"),
						},
					},
				},
			},
			err: errors.New("unsupported protocol \"http/invalid\""),
		},
		{
			name: "batch/otlp-grpc-exporter-no-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "grpc/protobuf",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-grpc-exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4317",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-grpc-exporter-no-scheme",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-grpc-invalid-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "grpc/protobuf",
							Endpoint:    " ",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			err: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "batch/otlp-grpc-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strPtr("invalid"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			err: errors.New("unsupported compression \"invalid\""),
		},
		{
			name: "batch/otlp-http-exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-http-exporter-with-path",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318/path/123",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-http-exporter-no-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "http/protobuf",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-http-exporter-no-scheme",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "batch/otlp-http-invalid-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "http/protobuf",
							Endpoint:    " ",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			err: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "batch/otlp-http-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						Otlp: &Otlp{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("invalid"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			err: errors.New("unsupported compression \"invalid\""),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSpanProcessor(context.Background(), tt.processor)
			assert.Equal(t, tt.err, err)
		})
	}
}
