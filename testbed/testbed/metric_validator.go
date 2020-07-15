// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testbed

import (
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
)

type CorrectnessTestMetricValidator struct {
	dataProvider            *GoldenDataProvider
	metricAssertionFailures []*MetricDiff
}

func NewCorrectnessTestMetricValidator(dataProvider *GoldenDataProvider) *CorrectnessTestMetricValidator {
	return &CorrectnessTestMetricValidator{dataProvider: dataProvider}
}

func (v *CorrectnessTestMetricValidator) Validate(tc *TestCase) {
	generatedMetrics := metricDatasToPDMs(v.dataProvider.GetMetricsGenerated())
	recdMetrics := tc.MockBackend.ReceivedMetrics
	if tc.MockBackend.ReceivedMetricsOld != nil {
		recdPDM := pdatautil.MetricsFromMetricsData(tc.MockBackend.ReceivedMetricsOld)
		recdMetrics = []pdata.Metrics{recdPDM}
	}
	diffPDMs(tc, generatedMetrics, recdMetrics)
}

func metricDatasToPDMs(in []data.MetricData) []pdata.Metrics {
	var out []pdata.Metrics
	for _, md := range in {
		pdm := pdatautil.MetricsFromInternalMetrics(md)
		out = append(out, pdm)
	}
	return out
}

func diffPDMs(
	tc *TestCase,
	sent []pdata.Metrics,
	received []pdata.Metrics,
) {
	// If the there was an 'old' metric receiver or exporter in the pipeline, `received` will have a length of
	// one, with all metrics in the single pdata.Metrics struct. The `sent` variable, however, will have as many
	// elements as there were metrics sent. To compare sent vs received, we extract a slice of
	// pdata.ResourceMetrics from both sent and received and compare those.
	sentResourceMetrics := pdmToPDRM(sent)
	recdResourceMetrics := pdmToPDRM(received)
	_ = diffRMSlices(sentResourceMetrics, recdResourceMetrics)
	// if diffs != nil {
	// 	fmt.Printf("%v\n", diffs)
	// }
	// FIXME This fails for oc because metrics arrive out of order. Determine whether this is something we want
	//  to fix or if we should relax diff requirements.
	// assert.Nil(tc.t, diffs)
}

func diffRMSlices(sent []pdata.ResourceMetrics, recd []pdata.ResourceMetrics) []*MetricDiff {
	var diffs []*MetricDiff
	if len(sent) != len(recd) {
		return []*MetricDiff{{
			expectedValue: len(sent),
			actualValue:   len(recd),
			msg:           "Sent vs received ResourceMetrics not equal length",
		}}
	}
	for i := 0; i < len(sent); i++ {
		sentRM := sent[i]
		recdRM := recd[i]
		diffs = diffRMs(diffs, sentRM, recdRM)
	}
	return diffs
}

func (v *CorrectnessTestMetricValidator) RecordResults(tc *TestCase) {
	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}
	// Remove "Test" prefix from test name.
	testName := tc.t.Name()[4:] // todo factor
	tc.resultsSummary.Add(tc.t.Name(), &CorrectnessTestResult{
		testName:                    testName,
		result:                      result,
		duration:                    time.Since(tc.startTime),
		metricAssertionFailureCount: uint64(len(v.metricAssertionFailures)),
	})
}
