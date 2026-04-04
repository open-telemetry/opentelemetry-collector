// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestValidateSemConvMetricURL(t *testing.T) {
	validURL := "https://github.com/open-telemetry/semantic-conventions/blob/v1.37.2/docs/system/system-metrics.md#metric-systemcputime"
	tests := []struct {
		name           string
		url            string
		semConvVersion string
		metricName     string
		wantErr        string
	}{
		{
			name:           "valid URL",
			url:            validURL,
			semConvVersion: "1.37.2",
			metricName:     "system.cpu.time",
		},
		{
			name:    "empty URL",
			wantErr: "url is empty",
		},
		{
			name:    "empty semConvVersion",
			url:     validURL,
			wantErr: "semConvVersion is empty",
		},
		{
			name:           "empty metricName",
			url:            validURL,
			semConvVersion: "1.37.2",
			wantErr:        "metricName is empty",
		},
		{
			name:           "wrong anchor",
			url:            validURL,
			semConvVersion: "1.37.2",
			metricName:     "default.metric",
			wantErr:        "invalid semantic-conventions URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSemConvMetricURL(tt.url, tt.semConvVersion, tt.metricName)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMetricInputTypeValidate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", false},
		{"string", false},
		{"int", true},
		{"double", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := MetricInputType{InputType: tt.input}.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHistogramValidate(t *testing.T) {
	tests := []struct {
		name    string
		h       Histogram
		wantErr string
	}{
		{
			name: "valid explicit default",
			h:    Histogram{},
		},
		{
			name: "valid explicit",
			h:    Histogram{Aggregation: HistogramAggregationExplicit},
		},
		{
			name: "valid exponential",
			h:    Histogram{Aggregation: HistogramAggregationExponential, MaxSize: 160, MaxScale: 10},
		},
		{
			name:    "invalid aggregation",
			h:       Histogram{Aggregation: "invalid"},
			wantErr: "invalid aggregation",
		},
		{
			name:    "exponential with boundaries",
			h:       Histogram{Aggregation: HistogramAggregationExponential, Boundaries: []float64{1.0, 2.0}},
			wantErr: "bucket_boundaries must not be set when aggregation is \"exponential\"",
		},
		{
			name:    "max_size on non-exponential",
			h:       Histogram{MaxSize: 5},
			wantErr: "max_size is only valid when aggregation is \"exponential\"",
		},
		{
			name:    "max_scale on non-exponential",
			h:       Histogram{MaxScale: 5},
			wantErr: "max_scale is only valid when aggregation is \"exponential\"",
		},
		{
			name:    "negative max_size",
			h:       Histogram{Aggregation: HistogramAggregationExponential, MaxSize: -1},
			wantErr: "max_size must be a positive integer",
		},
		{
			name:    "negative max_scale",
			h:       Histogram{Aggregation: HistogramAggregationExponential, MaxScale: -1},
			wantErr: "max_scale must be between 0 and 20",
		},
		{
			name:    "max_scale above 20",
			h:       Histogram{Aggregation: HistogramAggregationExponential, MaxScale: 21},
			wantErr: "max_scale must be between 0 and 20",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.h.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
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
		{&Histogram{Aggregation: HistogramAggregationExponential}, "ExponentialHistogram", true, false, "Histogram", false},
	} {
		assert.Equal(t, arg.wantType, arg.metricData.Type())
		assert.Equal(t, arg.wantHasAggregated, arg.metricData.HasAggregated())
		assert.Equal(t, arg.wantHasMonotonic, arg.metricData.HasMonotonic())
		assert.Equal(t, arg.wantInstrument, arg.metricData.Instrument())
		assert.Equal(t, arg.wantAsync, arg.metricData.IsAsync())
	}
}
