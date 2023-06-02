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
		reader any
		args   any
		err    error
	}{
		{
			name: "noreader",
			err:  errors.New("unsupported metric reader type: noreader"),
		},
		{
			name: "pull/prometheus-invalid-config-no-host",
			reader: telemetry.PullMetricReader{
				Exporter: telemetry.MetricExporter{
					"prometheus": telemetry.Prometheus{},
				},
			},
			err: errors.New("host must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: telemetry.PullMetricReader{
				Exporter: telemetry.MetricExporter{
					"prometheus": telemetry.Prometheus{
						Host: strPtr("locahost"),
					},
				},
			},
			err: errors.New("port must be specified"),
		},
		{
			name: "pull/prometheus-invalid-config-no-port",
			reader: telemetry.PullMetricReader{
				Exporter: telemetry.MetricExporter{
					"prometheus": telemetry.Prometheus{
						Host: strPtr("locahost"),
						Port: intPtr(8080),
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := InitMetricReader(context.Background(), tc.name, tc.reader, make(chan error))
			assert.Equal(t, tc.err, err)
		})
	}
}
