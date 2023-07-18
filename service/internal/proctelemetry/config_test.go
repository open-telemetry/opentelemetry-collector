// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/service/telemetry"
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
		reader telemetry.MetricReader
		args   any
		err    error
	}{
		{
			name: "noreader",
			err:  errors.New("unsupported metric reader type {<nil> <nil>}"),
		},
		{
			name: "pull prometheus invalid exporter",
			reader: telemetry.MetricReader{
				Pull: &telemetry.PullMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{},
					},
				},
			},
			err: errNoValidMetricExporter,
		},
		{
			name: "pull/prometheus-invalid-config-no-host",
			reader: telemetry.MetricReader{
				Pull: &telemetry.PullMetricReader{
					Exporter: telemetry.MetricExporter{
						Prometheus: &telemetry.Prometheus{},
					},
				},
			},
			err: errors.New("host must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: telemetry.MetricReader{
				Pull: &telemetry.PullMetricReader{
					Exporter: telemetry.MetricExporter{
						Prometheus: &telemetry.Prometheus{
							Host: strPtr("locahost"),
						},
					},
				},
			},
			err: errors.New("port must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: telemetry.MetricReader{
				Pull: &telemetry.PullMetricReader{
					Exporter: telemetry.MetricExporter{
						Prometheus: &telemetry.Prometheus{
							Host: strPtr("locahost"),
							Port: intPtr(8080),
						},
					},
				},
			},
		},
		{
			name: "periodic/invalid-exporter",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Prometheus: &telemetry.Prometheus{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{},
			},
			err: errNoValidMetricExporter,
		},
		{
			name: "periodic/console-exporter",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Console: telemetry.Console{},
					},
				},
			},
		},
		{
			name: "periodic/console-exporter-with-timeout-interval",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Interval: intPtr(10),
					Timeout:  intPtr(5),
					Exporter: telemetry.MetricExporter{
						Console: telemetry.Console{},
					},
				},
			},
		},
		{
			name: "periodic/otlp-exporter-invalid-protocol",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
							Protocol: *strPtr("http/invalid"),
						},
					},
				},
			},
			err: errors.New("unsupported protocol http/invalid"),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			name: "periodic/otlp-grpc-exporter-no-scheme",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
		},
		{
			name: "periodic/otlp-http-exporter",
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			reader: telemetry.MetricReader{
				Periodic: &telemetry.PeriodicMetricReader{
					Exporter: telemetry.MetricExporter{
						Otlp: &telemetry.OtlpMetric{
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
			_, _, err := InitMetricReader(context.Background(), tt.reader, make(chan error))
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestSpanProcessor(t *testing.T) {
	testCases := []struct {
		name      string
		processor telemetry.SpanProcessor
		args      any
		err       error
	}{
		{
			name: "no processor",
			err:  errors.New("unsupported span processor type {<nil> <nil>}"),
		},
		{
			name: "batch processor invalid exporter",
			processor: telemetry.SpanProcessor{
				Batch: &telemetry.BatchSpanProcessor{
					Exporter: telemetry.SpanExporter{},
				},
			},
			err: errNoValidSpanExporter,
		},
		{
			name: "batch processor invalid batch size console exporter",
			processor: telemetry.SpanProcessor{
				Batch: &telemetry.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(-1),
					Exporter: telemetry.SpanExporter{
						Console: telemetry.Console{},
					},
				},
			},
			err: errors.New("invalid batch size -1"),
		},
		{
			name: "batch processor invalid export timeout console exporter",
			processor: telemetry.SpanProcessor{
				Batch: &telemetry.BatchSpanProcessor{
					ExportTimeout: intPtr(-2),
					Exporter: telemetry.SpanExporter{
						Console: telemetry.Console{},
					},
				},
			},
			err: errors.New("invalid export timeout -2"),
		},
		{
			name: "batch processor invalid queue size console exporter",
			processor: telemetry.SpanProcessor{
				Batch: &telemetry.BatchSpanProcessor{
					MaxQueueSize: intPtr(-3),
					Exporter: telemetry.SpanExporter{
						Console: telemetry.Console{},
					},
				},
			},
			err: errors.New("invalid queue size -3"),
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: telemetry.SpanProcessor{
				Batch: &telemetry.BatchSpanProcessor{
					ScheduleDelay: intPtr(-4),
					Exporter: telemetry.SpanExporter{
						Console: telemetry.Console{},
					},
				},
			},
			err: errors.New("invalid schedule delay -4"),
		},
		{
			name: "batch processor console exporter",
			processor: telemetry.SpanProcessor{
				Batch: &telemetry.BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: telemetry.SpanExporter{
						Console: telemetry.Console{},
					},
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InitSpanProcessor(context.Background(), tt.processor)
			assert.Equal(t, tt.err, err)
		})
	}
}
