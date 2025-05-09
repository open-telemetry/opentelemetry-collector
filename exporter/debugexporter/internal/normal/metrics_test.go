// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestMarshalMetrics(t *testing.T) {
	tests := []struct {
		name     string
		input    pmetric.Metrics
		expected string
	}{
		{
			name:     "empty metrics",
			input:    pmetric.NewMetrics(),
			expected: "",
		},
		{
			name: "sum data point",
			input: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				metric.SetName("system.cpu.time")
				dataPoint := metric.SetEmptySum().DataPoints().AppendEmpty()
				dataPoint.SetDoubleValue(123.456)
				dataPoint.Attributes().PutStr("state", "user")
				dataPoint.Attributes().PutStr("cpu", "0")
				return metrics
			}(),
			expected: `ResourceMetrics #0
ScopeMetrics #0
system.cpu.time{state=user,cpu=0} 123.456
`,
		},
		{
			name: "data point with resource and scope attributes",
			input: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				resourceMetrics.SetSchemaUrl("https://opentelemetry.io/resource-schema-url")
				resourceMetrics.Resource().Attributes().PutStr("resourceKey1", "resourceValue1")
				resourceMetrics.Resource().Attributes().PutBool("resourceKey2", false)
				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				scopeMetrics.SetSchemaUrl("http://opentelemetry.io/scope-schema-url")
				scopeMetrics.Scope().SetName("scope-name")
				scopeMetrics.Scope().SetVersion("1.2.3")
				scopeMetrics.Scope().Attributes().PutStr("scopeKey1", "scopeValue1")
				scopeMetrics.Scope().Attributes().PutBool("scopeKey2", true)
				metric := scopeMetrics.Metrics().AppendEmpty()
				metric.SetName("system.cpu.time")
				dataPoint := metric.SetEmptySum().DataPoints().AppendEmpty()
				dataPoint.SetDoubleValue(123.456)
				dataPoint.Attributes().PutStr("state", "user")
				dataPoint.Attributes().PutStr("cpu", "0")
				return metrics
			}(),
			expected: `ResourceMetrics #0 [https://opentelemetry.io/resource-schema-url] resourceKey1=resourceValue1 resourceKey2=false
ScopeMetrics #0 scope-name@1.2.3 [http://opentelemetry.io/scope-schema-url] scopeKey1=scopeValue1 scopeKey2=true
system.cpu.time{state=user,cpu=0} 123.456
`,
		},
		{
			name: "gauge data point",
			input: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				metric.SetName("system.cpu.utilization")
				dataPoint := metric.SetEmptyGauge().DataPoints().AppendEmpty()
				dataPoint.SetDoubleValue(78.901234567)
				dataPoint.Attributes().PutStr("state", "free")
				dataPoint.Attributes().PutStr("cpu", "8")
				return metrics
			}(),
			expected: `ResourceMetrics #0
ScopeMetrics #0
system.cpu.utilization{state=free,cpu=8} 78.901234567
`,
		},
		{
			name: "histogram",
			input: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				metric.SetName("http.server.request.duration")
				dataPoint := metric.SetEmptyHistogram().DataPoints().AppendEmpty()
				dataPoint.Attributes().PutInt("http.response.status_code", 200)
				dataPoint.Attributes().PutStr("http.request.method", "GET")
				dataPoint.ExplicitBounds().FromRaw([]float64{0.125, 0.5, 1, 3})
				dataPoint.BucketCounts().FromRaw([]uint64{1324, 13, 0, 2, 1})
				dataPoint.SetCount(1340)
				dataPoint.SetSum(99.573)
				dataPoint.SetMin(0.017)
				dataPoint.SetMax(8.13)
				return metrics
			}(),
			expected: `ResourceMetrics #0
ScopeMetrics #0
http.server.request.duration{http.response.status_code=200,http.request.method=GET} count=1340 sum=99.573 min=0.017 max=8.13 le0.125=1324 le0.5=13 le1=0 le3=2 1
`,
		},
		{
			name: "exponential histogram",
			input: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				metric.SetName("http.server.request.duration")
				dataPoint := metric.SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
				dataPoint.Attributes().PutInt("http.response.status_code", 200)
				dataPoint.Attributes().PutStr("http.request.method", "GET")
				dataPoint.SetCount(1340)
				dataPoint.SetSum(99.573)
				dataPoint.SetMin(0.017)
				dataPoint.SetMax(8.13)
				return metrics
			}(),
			expected: `ResourceMetrics #0
ScopeMetrics #0
http.server.request.duration{http.response.status_code=200,http.request.method=GET} count=1340 sum=99.573 min=0.017 max=8.13
`,
		},
		{
			name: "summary",
			input: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				metric.SetName("summary")
				dataPoint := metric.SetEmptySummary().DataPoints().AppendEmpty()
				dataPoint.Attributes().PutInt("http.response.status_code", 200)
				dataPoint.Attributes().PutStr("http.request.method", "GET")
				dataPoint.SetCount(1340)
				dataPoint.SetSum(99.573)
				quantile := dataPoint.QuantileValues().AppendEmpty()
				quantile.SetQuantile(0.01)
				quantile.SetValue(15)
				return metrics
			}(),
			expected: `ResourceMetrics #0
ScopeMetrics #0
summary{http.response.status_code=200,http.request.method=GET} count=1340 sum=99.573000 q0.01=15
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := NewNormalMetricsMarshaler().MarshalMetrics(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}
