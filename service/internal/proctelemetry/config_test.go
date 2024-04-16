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
