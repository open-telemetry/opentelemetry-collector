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

func TestMetricValidate_DeprecatedWithoutDeprecatedStability(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelBeta,
			},
			Description: "test",
		},
		Deprecated: &Deprecated{
			Since: "1.0.0",
			Note:  "will be removed",
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
	assert.ErrorContains(t, err, "only allowed when stability level is 'deprecated'")
}

func TestMetricValidate_DeprecatedStabilityWithoutDeprecatedBlock(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelDeprecated,
			},
			Description: "test",
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
	assert.ErrorContains(t, err, "deprecated metrics must include deprecation metadata")
}

func TestMetricValidate_DeprecatedMissingSince(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelDeprecated,
			},
			Description: "test",
		},
		Deprecated: &Deprecated{
			Note: "will be removed",
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
	assert.ErrorContains(t, err, "deprecated.since is required")
}

func TestMetricValidate_DeprecatedMissingNote(t *testing.T) {
	m := &Metric{
		Signal: Signal{
			Stability: Stability{
				Level: component.StabilityLevelDeprecated,
			},
			Description: "test",
		},
		Deprecated: &Deprecated{
			Since: "1.0.0",
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
	assert.ErrorContains(t, err, "deprecated.note is required")
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

func TestStability_Unmarshal_WithFrom(t *testing.T) {
	// Test through the loader with a YAML file that includes the "from" field
	// This exercises the actual Unmarshal path with the "from" field
	md, err := LoadMetadata("testdata/with_stability_from.yaml")
	require.NoError(t, err)

	// Verify the metric was loaded and has the stability with "from" field
	testMetric, ok := md.Metrics["test.metric"]
	require.True(t, ok, "test.metric should exist")
	assert.Equal(t, component.StabilityLevelBeta, testMetric.Stability.Level)
	assert.Equal(t, "1.0.0", testMetric.Stability.From)
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
