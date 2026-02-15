// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestMetricAnchor(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		want       string
	}{
		{
			name:       "simple metric name",
			metricName: "system.cpu.time",
			want:       "metric-systemcputime",
		},
		{
			name:       "metric name with underscore",
			metricName: "system.disk.io_time",
			want:       "metric-systemdiskio_time",
		},
		{
			name:       "metric name with multiple underscores",
			metricName: "system.disk.weighted_io_time",
			want:       "metric-systemdiskweighted_io_time",
		},
		{
			name:       "metric name with uppercase",
			metricName: "System.CPU.Time",
			want:       "metric-systemcputime",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, metricAnchor(tt.metricName))
		})
	}
}

func TestValidateSemConvMetricURL(t *testing.T) {
	tests := []struct {
		name           string
		rawURL         string
		semConvVersion string
		metricName     string
		wantErr        bool
	}{
		{
			name:           "valid URL",
			rawURL:         "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemcputime",
			semConvVersion: "1.38.0",
			metricName:     "system.cpu.time",
		},
		{
			name:           "valid URL with underscore in metric name",
			rawURL:         "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemdiskio_time",
			semConvVersion: "1.38.0",
			metricName:     "system.disk.io_time",
		},
		{
			name:           "empty URL",
			rawURL:         "",
			semConvVersion: "1.38.0",
			metricName:     "system.cpu.time",
			wantErr:        true,
		},
		{
			name:           "empty semConvVersion",
			rawURL:         "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemcputime",
			semConvVersion: "",
			metricName:     "system.cpu.time",
			wantErr:        true,
		},
		{
			name:           "empty metricName",
			rawURL:         "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemcputime",
			semConvVersion: "1.38.0",
			metricName:     "",
			wantErr:        true,
		},
		{
			name:           "wrong version",
			rawURL:         "https://github.com/open-telemetry/semantic-conventions/blob/v1.37.0/docs/system/system-metrics.md#metric-systemcputime",
			semConvVersion: "1.38.0",
			metricName:     "system.cpu.time",
			wantErr:        true,
		},
		{
			name:           "wrong anchor",
			rawURL:         "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemcpuusage",
			semConvVersion: "1.38.0",
			metricName:     "system.cpu.time",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSemConvMetricURL(tt.rawURL, tt.semConvVersion, tt.metricName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricData(t *testing.T) {
	for _, arg := range []struct {
		metricData        MetricData
		wantType          string
		wantHasAggregated bool
		wantHasMonotonic  bool
		wantInstrument    string
		wantAsync         bool
	}{
		{&Gauge{}, "Gauge", false, false, "Gauge", false},
		{&Gauge{Async: true}, "Gauge", false, false, "ObservableGauge", true},
		{&Gauge{MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeInt}, Async: true}, "Gauge", false, false, "Int64ObservableGauge", true},
		{&Gauge{MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble}, Async: true}, "Gauge", false, false, "Float64ObservableGauge", true},
		{&Sum{}, "Sum", true, true, "UpDownCounter", false},
		{&Sum{Mono: Mono{true}}, "Sum", true, true, "Counter", false},
		{&Sum{Async: true}, "Sum", true, true, "ObservableUpDownCounter", true},
		{&Sum{MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeInt}, Async: true}, "Sum", true, true, "Int64ObservableUpDownCounter", true},
		{&Sum{MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble}, Async: true}, "Sum", true, true, "Float64ObservableUpDownCounter", true},
		{&Histogram{}, "Histogram", true, false, "Histogram", false},
	} {
		assert.Equal(t, arg.wantType, arg.metricData.Type())
		assert.Equal(t, arg.wantHasAggregated, arg.metricData.HasAggregated())
		assert.Equal(t, arg.wantHasMonotonic, arg.metricData.HasMonotonic())
		assert.Equal(t, arg.wantInstrument, arg.metricData.Instrument())
		assert.Equal(t, arg.wantAsync, arg.metricData.IsAsync())
	}
}
