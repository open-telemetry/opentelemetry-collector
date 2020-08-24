// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"

	"go.opentelemetry.io/collector/internal/data"
)

type traceMetricsCase struct {
	name string
	td   data.MetricData
	otlp []*otlpmetrics.ResourceMetrics
}

func generateAllMetricsTestCases() []traceMetricsCase {
	return []traceMetricsCase{
		{
			name: "empty",
			td:   GenerateMetricDataEmpty(),
			otlp: generateMetricOtlpEmpty(),
		},
		{
			name: "one-empty-resource-metrics",
			td:   GenerateMetricDataOneEmptyResourceMetrics(),
			otlp: generateMetricOtlpOneEmptyResourceMetrics(),
		},
		{
			name: "one-empty-one-nil-resource-metrics",
			td:   GenerateMetricDataOneEmptyOneNilResourceMetrics(),
			otlp: generateMetricOtlpOneEmptyOneNilResourceMetrics(),
		},
		{
			name: "no-libraries",
			td:   GenerateMetricDataNoLibraries(),
			otlp: generateMetricOtlpNoLibraries(),
		},
		{
			name: "one-empty-instrumentation-library",
			td:   GenerateMetricDataOneEmptyInstrumentationLibrary(),
			otlp: generateMetricOtlpOneEmptyInstrumentationLibrary(),
		},
		{
			name: "one-empty-one-nil-instrumentation-library",
			td:   GenerateMetricDataOneEmptyOneNilInstrumentationLibrary(),
			otlp: generateMetricOtlpOneEmptyOneNilInstrumentationLibrary(),
		},
		{
			name: "one-metric-no-resource",
			td:   GenerateMetricDataOneMetricNoResource(),
			otlp: generateMetricOtlpOneMetricNoResource(),
		},
		{
			name: "one-metric",
			td:   GenerateMetricDataOneMetric(),
			otlp: generateMetricOtlpOneMetric(),
		},
		{
			name: "two-metrics",
			td:   GenerateMetricDataTwoMetrics(),
			otlp: GenerateMetricOtlpTwoMetrics(),
		},
		{
			name: "one-metric-one-nil",
			td:   GenerateMetricDataOneMetricOneNil(),
			otlp: generateMetricOtlpOneMetricOneNil(),
		},
		{
			name: "one-metric-no-labels",
			td:   GenerateMetricDataOneMetricNoLabels(),
			otlp: generateMetricOtlpOneMetricNoLabels(),
		},
		{
			name: "one-metric-one-nil-point",
			td:   GenerateMetricDataOneMetricOneNilPoint(),
			otlp: generateMetricOtlpOneMetricOneNilPoint(),
		},
		{
			name: "all-types-no-data-points",
			td:   GenerateMetricDataAllTypesNoDataPoints(),
			otlp: generateMetricOtlpAllTypesNoDataPoints(),
		},
		{
			name: "counters-histogram-summary",
			td:   GenerateMetricDataWithCountersHistogramAndSummary(),
			otlp: generateMetricOtlpWithCountersHistogramAndSummary(),
		},
	}
}

func TestToFromOtlpMetrics(t *testing.T) {
	allTestCases := generateAllMetricsTestCases()
	// Ensure NumMetricTests gets updated.
	assert.EqualValues(t, NumMetricTests, len(allTestCases))
	for i := range allTestCases {
		test := allTestCases[i]
		t.Run(test.name, func(t *testing.T) {
			td := data.MetricDataFromOtlp(test.otlp)
			assert.EqualValues(t, test.td, td)
			otlp := data.MetricDataToOtlp(td)
			assert.EqualValues(t, test.otlp, otlp)
		})
	}
}

func TestToFromOtlpMetricsWithNils(t *testing.T) {
	md := GenerateMetricDataOneEmptyOneNilResourceMetrics()
	assert.EqualValues(t, 2, md.ResourceMetrics().Len())
	assert.False(t, md.ResourceMetrics().At(0).IsNil())
	assert.True(t, md.ResourceMetrics().At(1).IsNil())

	md = GenerateMetricDataOneEmptyOneNilInstrumentationLibrary()
	rs := md.ResourceMetrics().At(0)
	assert.EqualValues(t, 2, rs.InstrumentationLibraryMetrics().Len())
	assert.False(t, rs.InstrumentationLibraryMetrics().At(0).IsNil())
	assert.True(t, rs.InstrumentationLibraryMetrics().At(1).IsNil())

	md = GenerateMetricDataOneMetricOneNil()
	ilss := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	assert.EqualValues(t, 2, ilss.Metrics().Len())
	assert.False(t, ilss.Metrics().At(0).IsNil())
	assert.True(t, ilss.Metrics().At(1).IsNil())
}

func TestGenerateMetricDataManyMetricsSameResource(t *testing.T) {
	md := GenerateMetricDataManyMetricsSameResource(100)
	assert.EqualValues(t, 1, md.ResourceMetrics().Len())
	assert.EqualValues(t, 100, md.MetricCount())
}
