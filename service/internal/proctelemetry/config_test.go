// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry

import (
	"context"
	"errors"
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
			err: errors.New("no valid exporter"),
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
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := InitMetricReader(context.Background(), tt.reader, make(chan error))
			assert.Equal(t, tt.err, err)
		})
	}
}
