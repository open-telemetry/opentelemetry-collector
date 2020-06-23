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
	"reflect"
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
)

type CorrectnessTestMetricValidator struct {
	dataProvider            *GoldenDataProvider
	metricAssertionFailures []*MetricsAssertionFailure
}

func NewCorrectnessTestMetricValidator(dataProvider *GoldenDataProvider) *CorrectnessTestMetricValidator {
	return &CorrectnessTestMetricValidator{dataProvider: dataProvider}
}

func (v *CorrectnessTestMetricValidator) Validate(tc *TestCase) {
	receivedMetrics := tc.MockBackend.ReceivedMetrics
	generatedMetrics := v.dataProvider.GetMetricsGenerated()
	var pdGeneratedMetrics []pdata.Metrics
	for _, md := range generatedMetrics {
		pdm := pdatautil.MetricsFromInternalMetrics(md)
		pdGeneratedMetrics = append(pdGeneratedMetrics, pdm)
	}
	v.diffMetrics(pdGeneratedMetrics, receivedMetrics)
}

func (v *CorrectnessTestMetricValidator) diffMetrics(sent []pdata.Metrics, received []pdata.Metrics) {
	eq := reflect.DeepEqual(sent, received)
	if !eq {
		v.metricAssertionFailures = append(v.metricAssertionFailures, &MetricsAssertionFailure{
			expected: sent,
			actual:   received,
		})
	}
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
