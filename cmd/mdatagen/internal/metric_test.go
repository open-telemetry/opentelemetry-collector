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

func TestMetricValidate(t *testing.T) {
	unit := "s"
	validGauge := &Gauge{MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble}}

	tests := []struct {
		name           string
		metric         Metric
		metricName     MetricName
		semConvVersion string
		wantErr        string
	}{
		{
			name: "missing stability level",
			metric: Metric{
				Signal: Signal{Description: "A metric"},
				Unit:   &unit,
				Gauge:  validGauge,
			},
			metricName: "my.metric",
			wantErr:    "missing required field: `stability.level`",
		},
		{
			name: "stability must be deprecated when deprecated field set",
			metric: Metric{
				Signal: Signal{
					Description: "A metric",
					Stability:   component.StabilityLevelBeta,
				},
				Unit:       &unit,
				Gauge:      validGauge,
				Deprecated: &Deprecated{Since: "1.0.0"},
			},
			metricName: "my.metric",
			wantErr:    "`stability` must be `deprecated` when specifying a `deprecated` field",
		},
		{
			name: "invalid semantic convention URL",
			metric: Metric{
				Signal: Signal{
					Description: "A metric",
					Stability:   component.StabilityLevelBeta,
					SemanticConvention: &SemanticConvention{
						// anchor points to a different metric name than the metric being validated
						SemanticConventionRef: "https://github.com/open-telemetry/semantic-conventions/blob/v1.37.2/docs/system/system-metrics.md#metric-systemcputime",
					},
				},
				Unit:  &unit,
				Gauge: validGauge,
			},
			metricName:     "default.metric",
			semConvVersion: "1.37.2",
			wantErr:        "invalid semantic-conventions URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metric.validate(tt.metricName, tt.semConvVersion)
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
	} {
		assert.Equal(t, arg.wantType, arg.metricData.Type())
		assert.Equal(t, arg.wantHasAggregated, arg.metricData.HasAggregated())
		assert.Equal(t, arg.wantHasMonotonic, arg.metricData.HasMonotonic())
		assert.Equal(t, arg.wantInstrument, arg.metricData.Instrument())
		assert.Equal(t, arg.wantAsync, arg.metricData.IsAsync())
	}
}
