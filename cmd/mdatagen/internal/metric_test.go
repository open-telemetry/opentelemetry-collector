// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

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

func TestMetricValidate(t *testing.T) {
	tests := []struct {
		name    string
		metric  *Metric
		wantErr string
	}{
		{
			name: "missing metric type",
			metric: &Metric{
				Signal: Signal{
					Stability:   Stability{Level: component.StabilityLevelBeta},
					Description: "test",
				},
				Unit: ptr("1"),
			},
			wantErr: "missing metric type key",
		},
		{
			name: "multiple metric types",
			metric: &Metric{
				Signal: Signal{
					Stability:   Stability{Level: component.StabilityLevelBeta},
					Description: "test",
				},
				Unit: ptr("1"),
				Sum: &Sum{
					MetricValueType: MetricValueType{ValueType: pmetric.NumberDataPointValueTypeInt},
				},
				Gauge: &Gauge{
					MetricValueType: MetricValueType{ValueType: pmetric.NumberDataPointValueTypeInt},
				},
			},
			wantErr: "more than one metric type keys",
		},
		{
			name: "valid metric",
			metric: &Metric{
				Signal: Signal{
					Stability:   Stability{Level: component.StabilityLevelBeta},
					Description: "test",
				},
				Unit: ptr("1"),
				Sum: &Sum{
					MetricValueType: MetricValueType{ValueType: pmetric.NumberDataPointValueTypeInt},
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metric.validate("test.metric", "1.0.0")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
