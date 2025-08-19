// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMetricsText(t *testing.T) {
	tests := []struct {
		name string
		in   pmetric.Metrics
		out  string
	}{
		{
			name: "empty_metrics",
			in:   pmetric.NewMetrics(),
			out:  "empty.out",
		},
		{
			name: "metrics_with_all_types",
			in:   testdata.GenerateMetricsAllTypes(),
			out:  "metrics_with_all_types.out",
		},
		{
			name: "two_metrics",
			in:   testdata.GenerateMetrics(2),
			out:  "two_metrics.out",
		},
		{
			name: "invalid_metric_type",
			in:   testdata.GenerateMetricsMetricTypeInvalid(),
			out:  "invalid_metric_type.out",
		},
		{
			name: "metrics_with_entity_refs",
			in:   generateMetricsWithEntityRefs(),
			out:  "metrics_with_entity_refs.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextMetricsMarshaler().MarshalMetrics(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "metrics", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func generateMetricsWithEntityRefs() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()

	setupResourceWithEntityRefs(rm.Resource())

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test-scope")
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	metric.SetDescription("A test metric")
	metric.SetUnit("1")

	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetDoubleValue(123.45)
	dp.Attributes().PutStr("test.attribute", "test-value")

	return md
}
