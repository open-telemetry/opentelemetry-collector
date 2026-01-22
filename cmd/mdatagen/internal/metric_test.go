// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
<<<<<<< HEAD
=======
	"go.opentelemetry.io/collector/confmap"
>>>>>>> db222d477 (mdatagen: add stability tests)
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

<<<<<<< HEAD
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
=======
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
>>>>>>> db222d477 (mdatagen: add stability tests)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
<<<<<<< HEAD
			err := tt.metric.validate("test.metric", "1.0.0")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
=======
			got := tt.stability.String()
			assert.Equal(t, tt.want, got)
>>>>>>> db222d477 (mdatagen: add stability tests)
		})
	}
}

<<<<<<< HEAD
func ptr[T any](v T) *T {
	return &v
=======
func TestStability_Unmarshal_WithoutFrom(t *testing.T) {
	parser := confmap.NewFromStringMap(map[string]any{
		"level": "beta",
	})

	var s Stability
	err := s.Unmarshal(parser)
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelBeta, s.Level)
	assert.Empty(t, s.From)
>>>>>>> db222d477 (mdatagen: add stability tests)
}

func TestMetricValidate_MissingMetricType(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
		},
		Unit: ptr("1"),
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing metric type key")
}

func TestMetricValidate_MultipleMetricTypes(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
		},
		Unit: ptr("1"),
		Sum: &Sum{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
		},
		Gauge: &Gauge{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "more than one metric type keys")
}

func TestMetricValidate_MissingDescription(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "",
		},
		Unit: ptr("1"),
		Sum: &Sum{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing metric description")
}

func TestMetricValidate_MissingUnit(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
		},
		Unit: nil,
		Sum: &Sum{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing metric unit")
}

func TestMetricValidate_InvalidInputType_Sum(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
		},
		Unit: ptr("1"),
		Sum: &Sum{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
			MetricInputType: MetricInputType{
				InputType: "invalid",
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid `input_type` value")
}

func TestMetricValidate_InvalidInputType_Gauge(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
		},
		Unit: ptr("1"),
		Gauge: &Gauge{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
			MetricInputType: MetricInputType{
				InputType: "invalid",
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid `input_type` value")
}

func TestMetricValidate_InvalidSemanticConventionURL(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
			SemanticConvention: &SemanticConvention{
				SemanticConventionRef: "https://invalid-url.com",
			},
		},
		Unit: ptr("1"),
		Sum: &Sum{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid semantic-conventions URL")
}

func TestMetricValidate_ValidSemanticConventionURL(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
			SemanticConvention: &SemanticConvention{
				SemanticConventionRef: "https://github.com/open-telemetry/semantic-conventions/blob/v1.0.0/docs/metrics/testmetric#metric-testmetric",
			},
		},
		Unit: ptr("1"),
		Sum: &Sum{
			MetricValueType: MetricValueType{
				ValueType: pmetric.NumberDataPointValueTypeInt,
			},
		},
	}

	err := m.validate("test.metric", "1.0.0")
	require.NoError(t, err)
}

func ptr[T any](v T) *T {
	return &v
}
