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

package internaldata

import (
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestOCToMetricData(t *testing.T) {
	tests := []struct {
		name     string
		oc       consumerdata.MetricsData
		internal data.MetricData
	}{
		{
			name:     "empty",
			oc:       consumerdata.MetricsData{},
			internal: testdata.GenerateMetricDataEmpty(),
		},

		{
			name: "one-empty-resource-metrics",
			oc: consumerdata.MetricsData{
				Node:     &occommon.Node{},
				Resource: &ocresource.Resource{},
			},
			internal: wrapMetricsWithEmptyResource(testdata.GenerateMetricDataOneEmptyResourceMetrics()),
		},

		{
			name:     "no-libraries",
			oc:       generateOCTestDataNoMetrics(),
			internal: testdata.GenerateMetricDataNoLibraries(),
		},

		{
			name:     "all-types-no-points",
			oc:       generateOCTestDataNoPoints(),
			internal: testdata.GenerateMetricDataAllTypesNoDataPoints(),
		},

		{
			name:     "one-metric-no-labels",
			oc:       generateOCTestDataNoLabels(),
			internal: testdata.GenerateMetricDataOneMetricNoLabels(),
		},

		{
			name:     "one-metric",
			oc:       generateOCTestDataMetricsOneMetric(),
			internal: testdata.GenerateMetricDataOneMetric(),
		},

		{
			name:     "one-metric-one-nil",
			oc:       generateOCTestDataMetricsOneMetricOneNil(),
			internal: testdata.GenerateMetricDataOneMetric(),
		},

		{
			name:     "one-metric-one-nil-timeseries",
			oc:       generateOCTestDataMetricsOneMetricOneNilTimeseries(),
			internal: testdata.GenerateMetricDataOneMetric(),
		},

		{
			name:     "one-metric-one-nil-point",
			oc:       generateOCTestDataMetricsOneMetricOneNilPoint(),
			internal: testdata.GenerateMetricDataOneMetric(),
		},

		{
			name: "sample-metric",
			oc: consumerdata.MetricsData{
				Resource: generateOCTestResource(),
				Metrics: []*ocmetrics.Metric{
					generateOCTestMetricInt(),
					generateOCTestMetricDouble(),
					generateOCTestMetricHistogram(),
					generateOCTestMetricSummary(),
				},
			},
			internal: testdata.GenerateMetricDataWithCountersHistogramAndSummary(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := OCToMetricData(test.oc)
			assert.EqualValues(t, test.internal, got)

			ocslice := []consumerdata.MetricsData{
				test.oc,
				test.oc,
			}
			wantSlice := data.NewMetricData()
			// Double the ResourceMetrics only if not empty.
			if test.internal.ResourceMetrics().Len() != 0 {
				test.internal.Clone().ResourceMetrics().MoveAndAppendTo(wantSlice.ResourceMetrics())
				test.internal.Clone().ResourceMetrics().MoveAndAppendTo(wantSlice.ResourceMetrics())
			}
			gotSlice := OCSliceToMetricData(ocslice)
			assert.EqualValues(t, wantSlice, gotSlice)
		})
	}
}

// TODO: Try to avoid unnecessary Resource object allocation.
func wrapMetricsWithEmptyResource(md data.MetricData) data.MetricData {
	md.ResourceMetrics().At(0).Resource().InitEmpty()
	return md
}

func BenchmarkMetricIntOCToInternal(b *testing.B) {
	ocMetric := consumerdata.MetricsData{
		Resource: generateOCTestResource(),
		Metrics: []*ocmetrics.Metric{
			generateOCTestMetricInt(),
			generateOCTestMetricInt(),
			generateOCTestMetricInt(),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		OCToMetricData(ocMetric)
	}
}

func BenchmarkMetricDoubleOCToInternal(b *testing.B) {
	ocMetric := consumerdata.MetricsData{
		Resource: generateOCTestResource(),
		Metrics: []*ocmetrics.Metric{
			generateOCTestMetricDouble(),
			generateOCTestMetricDouble(),
			generateOCTestMetricDouble(),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		OCToMetricData(ocMetric)
	}
}

func BenchmarkMetricHistogramOCToInternal(b *testing.B) {
	ocMetric := consumerdata.MetricsData{
		Resource: generateOCTestResource(),
		Metrics: []*ocmetrics.Metric{
			generateOCTestMetricHistogram(),
			generateOCTestMetricHistogram(),
			generateOCTestMetricHistogram(),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		OCToMetricData(ocMetric)
	}
}

func BenchmarkMetricSummaryOCToInternal(b *testing.B) {
	ocMetric := consumerdata.MetricsData{
		Resource: generateOCTestResource(),
		Metrics: []*ocmetrics.Metric{
			generateOCTestMetricSummary(),
			generateOCTestMetricSummary(),
			generateOCTestMetricSummary(),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		OCToMetricData(ocMetric)
	}
}

func generateOCTestResource() *ocresource.Resource {
	return &ocresource.Resource{
		Labels: map[string]string{
			"resource-attr": "resource-attr-val-1",
		},
	}
}
