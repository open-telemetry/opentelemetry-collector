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

package metrics

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/testbed/correctness"
	"go.opentelemetry.io/collector/testbed/testbed"
)

// tests with the prefix "TestHarness_" get run in the "correctness-metrics" ci job
func TestHarness_MetricsGoldenData(t *testing.T) {
	tests, err := correctness.LoadPictOutputPipelineDefs(
		"testdata/generated_pict_pairs_metrics_pipeline.txt",
	)
	require.NoError(t, err)

	res := results{}
	res.Init("results")
	for _, test := range tests {
		test.TestName = fmt.Sprintf("%s-%s", test.Receiver, test.Exporter)
		test.DataSender = correctness.ConstructMetricsSender(t, test.Receiver)
		test.DataReceiver = correctness.ConstructReceiver(t, test.Exporter)
		t.Run(test.TestName, func(t *testing.T) {
			r := testWithMetricsGoldenDataset(
				t,
				test.DataSender.(testbed.MetricDataSender),
				test.DataReceiver,
			)
			res.Add("", r)
		})
	}
	res.Save()
}

func testWithMetricsGoldenDataset(
	t *testing.T,
	sender testbed.MetricDataSender,
	receiver testbed.DataReceiver,
) result {
	mds := getTestMetrics(t)
	accumulator := newDiffAccumulator()
	h := newTestHarness(
		t,
		newMetricSupplier(mds),
		newMetricsReceivedIndex(mds),
		sender,
		accumulator,
	)
	tc := newCorrectnessTestCase(t, sender, receiver, h)

	tc.startTestbedReceiver()
	tc.startCollector()
	tc.startTestbedSender()

	tc.sendFirstMetric()
	tc.waitForAllMetrics()

	tc.stopTestbedReceiver()
	tc.stopCollector()

	r := result{
		testName:   t.Name(),
		testResult: "PASS",
		numDiffs:   accumulator.numDiffs,
	}
	if accumulator.numDiffs > 0 {
		r.testResult = "FAIL"
		t.Fail()
	}
	return r
}

func getTestMetrics(t *testing.T) []pdata.Metrics {
	const file = "../../../internal/goldendataset/testdata/generated_pict_pairs_metrics.txt"
	mds, err := goldendataset.GenerateMetricDatas(file)
	require.NoError(t, err)
	return mds
}

type diffAccumulator struct {
	numDiffs int
}

var _ diffConsumer = (*diffAccumulator)(nil)

func newDiffAccumulator() *diffAccumulator {
	return &diffAccumulator{}
}

func (d *diffAccumulator) accept(metricName string, diffs []*MetricDiff) {
	if len(diffs) > 0 {
		d.numDiffs++
		log.Printf("Found diffs for [%v]\n%v", metricName, diffs)
	}
}
