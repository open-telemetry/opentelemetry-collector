package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
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

func ptr[T any](v T) *T {
	return &v
}
