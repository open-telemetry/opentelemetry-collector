// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
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

func TestStability_String(t *testing.T) {
	tests := []struct {
		name      string
		stability Stability
		want      string
	}{
		{
			name: "undefined level",
			stability: Stability{
				Level: component.StabilityLevelUndefined,
			},
			want: "",
		},
		{
			name: "stable level",
			stability: Stability{
				Level: component.StabilityLevelStable,
			},
			want: "",
		},
		{
			name: "beta level",
			stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			want: " [Beta]",
		},
		{
			name: "alpha level",
			stability: Stability{
				Level: component.StabilityLevelAlpha,
			},
			want: " [Alpha]",
		},
		{
			name: "deprecated level",
			stability: Stability{
				Level: component.StabilityLevelDeprecated,
			},
			want: " [Deprecated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stability.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStability_Unmarshal_WithoutFrom(t *testing.T) {
	parser := confmap.NewFromStringMap(map[string]any{
		"level": "beta",
	})

	var s Stability
	err := s.Unmarshal(parser)
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelBeta, s.Level)
	assert.Empty(t, s.From)
}
