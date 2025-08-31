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

	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
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
			got, err := NewTextMetricsMarshaler(internal.NewDefaultOutputConfig()).MarshalMetrics(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "metrics", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func TestMetricsWithOutputConfig(t *testing.T) {
	tests := []struct {
		name   string
		in     pmetric.Metrics
		out    string
		config internal.OutputConfig
	}{
		{
			name: "marshal_metrics_with_attributes_filter_include",
			in:   generateBasicMetrics(),
			out:  "metric_with_attributes_filter_include.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_metrics_with_attributes_filter_exclude",
			in:   generateBasicMetrics(),
			out:  "metric_with_attributes_filter_exclude.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Exclude: []string{"attribute.remove"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Exclude: []string{"attribute.remove"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Exclude: []string{"attribute.remove"},
					},
				},
			},
		},
		{
			name: "marshal_metrics_with_record_disabled",
			in:   generateBasicMetrics(),
			out:  "metric_with_attributes_filter_with_record_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_metrics_with_scope_disabled",
			in:   generateBasicMetrics(),
			out:  "metric_with_attributes_filter_with_scope_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_metrics_with_resource_disabled",
			in:   generateBasicMetrics(),
			out:  "metric_with_attributes_filter_with_resource_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_metrics_with_attributes_disabled",
			in:   generateBasicMetrics(),
			out:  "metric_with_attributes_filter_with_attributes_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: false,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: false,
						Include: []string{"attribute.keep"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: false,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextMetricsMarshaler(tt.config).MarshalMetrics(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "metrics", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func generateBasicMetrics() pmetric.Metrics {
	md := testdata.GenerateMetricsAllTypes()
	rm := md.ResourceMetrics().At(0)
	rm.Resource().Attributes().PutStr("attribute.keep", "resource-keep")
	rm.Resource().Attributes().PutStr("attribute.remove", "resource-remove")
	sm := rm.ScopeMetrics().At(0)
	sm.Scope().Attributes().PutStr("attribute.keep", "scope-keep")
	sm.Scope().Attributes().PutStr("attribute.remove", "scope-remove")

	for i := 0; i < sm.Metrics().Len(); i++ {
		m := sm.Metrics().At(i)
		switch m.Type() {
		case pmetric.MetricTypeGauge:
			dps := m.Gauge().DataPoints()
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				dp.Attributes().PutStr("attribute.keep", "metric-keep")
				dp.Attributes().PutStr("attribute.remove", "metric-remove")
			}
		case pmetric.MetricTypeSum:
			dps := m.Sum().DataPoints()
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				dp.Attributes().PutStr("attribute.keep", "metric-keep")
				dp.Attributes().PutStr("attribute.remove", "metric-remove")
			}
		case pmetric.MetricTypeHistogram:
			dps := m.Histogram().DataPoints()
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				dp.Attributes().PutStr("attribute.keep", "metric-keep")
				dp.Attributes().PutStr("attribute.remove", "metric-remove")
			}
		case pmetric.MetricTypeExponentialHistogram:
			dps := m.ExponentialHistogram().DataPoints()
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				dp.Attributes().PutStr("attribute.keep", "metric-keep")
				dp.Attributes().PutStr("attribute.remove", "metric-remove")
			}
		case pmetric.MetricTypeSummary:
			dps := m.Summary().DataPoints()
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				dp.Attributes().PutStr("attribute.keep", "metric-keep")
				dp.Attributes().PutStr("attribute.remove", "metric-remove")
			}
		}
	}
	return md
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
