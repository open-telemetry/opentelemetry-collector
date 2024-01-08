// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/config"
)

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func TestMetricReader(t *testing.T) {
	testCases := []struct {
		name   string
		reader config.MetricReader
		args   any
		err    error
	}{
		{
			name: "noreader",
			err:  errors.New("unsupported metric reader type {<nil> <nil>}"),
		},
		{
			name: "pull prometheus invalid exporter",
			reader: config.MetricReader{
				Pull: &config.PullMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{},
					},
				},
			},
			err: errNoValidMetricExporter,
		},
		{
			name: "pull/prometheus-invalid-config-no-host",
			reader: config.MetricReader{
				Pull: &config.PullMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{},
					},
				},
			},
			err: errors.New("host must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: config.MetricReader{
				Pull: &config.PullMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{
							Host: strPtr("locahost"),
						},
					},
				},
			},
			err: errors.New("port must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: config.MetricReader{
				Pull: &config.PullMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{
							Host: strPtr("locahost"),
							Port: intPtr(8080),
						},
					},
				},
			},
		},
		{
			name: "periodic/invalid-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{
							Host: strPtr("locahost"),
							Port: intPtr(8080),
						},
					},
				},
			},
			err: errNoValidMetricExporter,
		},
		{
			name: "periodic/no-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{},
			},
			err: errNoValidMetricExporter,
		},
		{
			name: "periodic/console-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						Console: config.Console{},
					},
				},
			},
		},
		{
			name: "periodic/console-exporter-with-timeout-interval",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Interval: intPtr(10),
					Timeout:  intPtr(5),
					Exporter: config.MetricExporter{
						Console: config.Console{},
					},
				},
			},
		},
		{
			name: "periodic/otlp-exporter-invalid-protocol",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol: *strPtr("http/invalid"),
						},
					},
				},
			},
			err: errors.New("unsupported protocol http/invalid"),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-grpc-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4317",
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
			name: "periodic/otlp-grpc-exporter-no-scheme",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-grpc-invalid-endpoint",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-grpc-invalid-compression",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-http-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-http-exporter-with-path",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-http-exporter-no-endpoint",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-http-exporter-no-scheme",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-http-invalid-endpoint",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			name: "periodic/otlp-http-invalid-compression",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
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
			reader, server, err := InitMetricReader(context.Background(), tt.reader, make(chan error))
			defer func() {
				if reader != nil {
					assert.NoError(t, reader.Shutdown(context.Background()))
				}
				if server != nil {
					assert.NoError(t, server.Shutdown(context.Background()))
				}
			}()
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestSpanProcessor(t *testing.T) {
	testCases := []struct {
		name      string
		processor config.SpanProcessor
		args      any
		err       error
	}{
		{
			name: "no processor",
			err:  errors.New("unsupported span processor type {<nil> <nil>}"),
		},
		{
			name: "batch processor invalid exporter",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					Exporter: config.SpanExporter{},
				},
			},
			err: errNoValidSpanExporter,
		},
		{
			name: "batch processor invalid batch size console exporter",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(-1),
					Exporter: config.SpanExporter{
						Console: config.Console{},
					},
				},
			},
			err: errors.New("invalid batch size -1"),
		},
		{
			name: "batch processor invalid export timeout console exporter",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					ExportTimeout: intPtr(-2),
					Exporter: config.SpanExporter{
						Console: config.Console{},
					},
				},
			},
			err: errors.New("invalid export timeout -2"),
		},
		{
			name: "batch processor invalid queue size console exporter",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxQueueSize: intPtr(-3),
					Exporter: config.SpanExporter{
						Console: config.Console{},
					},
				},
			},
			err: errors.New("invalid queue size -3"),
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					ScheduleDelay: intPtr(-4),
					Exporter: config.SpanExporter{
						Console: config.Console{},
					},
				},
			},
			err: errors.New("invalid schedule delay -4"),
		},
		{
			name: "batch processor console exporter",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						Console: config.Console{},
					},
				},
			},
		},
		{
			name: "batch/otlp-exporter-invalid-protocol",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
							Protocol: *strPtr("http/invalid"),
						},
					},
				},
			},
			err: errors.New("unsupported protocol \"http/invalid\""),
		},
		{
			name: "batch/otlp-grpc-exporter-no-endpoint",
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor: config.SpanProcessor{
				Batch: &config.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: config.SpanExporter{
						OTLP: &config.OTLP{
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
			processor, err := InitSpanProcessor(context.Background(), tt.processor)
			defer func() {
				if processor != nil {
					assert.NoError(t, processor.Shutdown(context.Background()))
				}
			}()
			assert.Equal(t, tt.err, err)
		})
	}
}
