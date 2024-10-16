// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelinit

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func TestMetricReader(t *testing.T) {
	consoleExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	require.NoError(t, err)
	ctx := context.Background()
	otlpGRPCExporter, err := otlpmetricgrpc.New(ctx)
	require.NoError(t, err)
	otlpHTTPExporter, err := otlpmetrichttp.New(ctx)
	require.NoError(t, err)
	promExporter, err := otelprom.New()
	require.NoError(t, err)

	testCases := []struct {
		name       string
		reader     config.MetricReader
		args       any
		wantErr    error
		wantReader sdkmetric.Reader
	}{
		{
			name:    "noreader",
			wantErr: errors.New("unsupported metric reader type {<nil> <nil>}"),
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
			wantErr: errNoValidMetricExporter,
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
			wantErr: errors.New("host must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: config.MetricReader{
				Pull: &config.PullMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{
							Host: strPtr("localhost"),
						},
					},
				},
			},
			wantErr: errors.New("port must be specified"),
		},
		{
			name: "pull/prometheus-valid",
			reader: config.MetricReader{
				Pull: &config.PullMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{
							Host: strPtr("localhost"),
							Port: intPtr(8080),
						},
					},
				},
			},
			wantReader: promExporter,
		},
		{
			name: "periodic/invalid-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						Prometheus: &config.Prometheus{
							Host: strPtr("localhost"),
							Port: intPtr(8080),
						},
					},
				},
			},
			wantErr: errNoValidMetricExporter,
		},
		{
			name: "periodic/no-exporter",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{},
			},
			wantErr: errNoValidMetricExporter,
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
			wantReader: sdkmetric.NewPeriodicReader(consoleExporter),
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
			wantReader: sdkmetric.NewPeriodicReader(consoleExporter),
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
			wantErr: errors.New("unsupported protocol http/invalid"),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
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
			wantErr: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
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
			wantErr: errors.New("unsupported compression \"invalid\""),
		},
		{
			name: "periodic/otlp-grpc-delta-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("delta"),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-cumulative-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("cumulative"),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-lowmemory-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("lowmemory"),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-invalid-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("invalid"),
						},
					},
				},
			},
			wantErr: errors.New("unsupported temporality preference \"invalid\""),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
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
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
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
			wantErr: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
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
			wantErr: errors.New("unsupported compression \"invalid\""),
		},
		{
			name: "periodic/otlp-http-cumulative-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("cumulative"),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-lowmemory-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("lowmemory"),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-delta-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("delta"),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-invalid-temporality",
			reader: config.MetricReader{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.MetricExporter{
						OTLP: &config.OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: strPtr("invalid"),
						},
					},
				},
			},
			wantErr: errors.New("unsupported temporality preference \"invalid\""),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			gotReader, server, err := InitMetricReader(context.Background(), tt.reader, make(chan error), &sync.WaitGroup{})

			defer func() {
				if gotReader != nil {
					require.NoError(t, gotReader.Shutdown(context.Background()))
				}
				if server != nil {
					assert.NoError(t, server.Shutdown(context.Background()))
				}
			}()

			assert.Equal(t, tt.wantErr, err)

			if tt.wantReader == nil {
				assert.Nil(t, gotReader)
			} else {
				assert.Equal(t, reflect.TypeOf(tt.wantReader), reflect.TypeOf(gotReader))

				if reflect.TypeOf(tt.wantReader).String() == "*metric.PeriodicReader" {
					wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantReader)).FieldByName("exporter").Elem().Type()
					gotExporterType := reflect.Indirect(reflect.ValueOf(gotReader)).FieldByName("exporter").Elem().Type()
					assert.Equal(t, wantExporterType, gotExporterType)
				}
			}
		})
	}
}
