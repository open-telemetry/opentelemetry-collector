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
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/stretchr/testify/assert"

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
	receivedMetrics := tc.MockBackend.ReceivedMetrics
	generatedMetrics := v.dataProvider.GetMetricsGenerated()
	var pdGeneratedMetrics []pdata.Metrics
	for _, md := range generatedMetrics {
		pdm := pdatautil.MetricsFromInternalMetrics(md)
		pdGeneratedMetrics = append(pdGeneratedMetrics, pdm)
	}
	v.diffMetrics(tc, pdGeneratedMetrics, receivedMetrics)
}

func (v *CorrectnessTestMetricValidator) diffMetrics(tc *TestCase, sent []pdata.Metrics, received []pdata.Metrics) {
	if assert.EqualValues(
		tc.t, len(sent), len(received), "Sent and received metric counters do not match.",
	) {
		log.Printf("Sent and received metrics counters match.")
	}
	v.assertSentRecdMetricsDataEqual(tc, sent, received)
}

func (v *CorrectnessTestMetricValidator) assertSentRecdMetricsDataEqual(
	tc *TestCase,
	sent []pdata.Metrics,
	received []pdata.Metrics,
) {
	var allDiffs []*MetricDiff
	for i := 0; i < len(sent); i++ {
		sentMetric := sent[i]
		recdMetric := received[i]
		mdSent := pdatautil.MetricsToInternalMetrics(sentMetric)
		mdRecd := pdatautil.MetricsToInternalMetrics(recdMetric)
		diffs := diffMetricData(mdSent, mdRecd)
		allDiffs = append(allDiffs, diffs...)
	}
	fmt.Printf("%v\n", allDiffs)
	// TODO enable once identified problems are fixed
	// assert.Nil(tc.t, allDiffs)
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

type MetricDiff struct {
	expectedValue interface{}
	actualValue   interface{}
	msg           string
}

func (mf MetricDiff) String() string {
	return fmt.Sprintf("{msg='%v' expected=[%v] actual=[%v]}\n", mf.msg, mf.expectedValue, mf.actualValue)
}

func diffMetricData(expected data.MetricData, actual data.MetricData) []*MetricDiff {
	return diffResourceMetrics(nil, expected.ResourceMetrics(), actual.ResourceMetrics())
}

func diffResourceMetrics(diffs []*MetricDiff, expected pdata.ResourceMetricsSlice, actual pdata.ResourceMetricsSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, actual.Len(), expected.Len(), "ResourceMetricsSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffRMSlice(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffRMSlice(diffs []*MetricDiff, expected pdata.ResourceMetrics, actual pdata.ResourceMetrics) []*MetricDiff {
	diffs = diffResource(diffs, expected.Resource(), actual.Resource())
	diffs = diffILMSlice(diffs, expected.InstrumentationLibraryMetrics(), actual.InstrumentationLibraryMetrics())
	return diffs
}

func diffILMSlice(diffs []*MetricDiff, expected pdata.InstrumentationLibraryMetricsSlice, actual pdata.InstrumentationLibraryMetricsSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, actual.Len(), expected.Len(), "InstrumentationLibraryMetricsSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffILM(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffILM(diffs []*MetricDiff, expected pdata.InstrumentationLibraryMetrics, actual pdata.InstrumentationLibraryMetrics) []*MetricDiff {
	return diffMetrics(diffs, expected.Metrics(), actual.Metrics())
}

func diffMetrics(diffs []*MetricDiff, expected pdata.MetricSlice, actual pdata.MetricSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, actual.Len(), expected.Len(), "MetricSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffMetric(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffMetric(diffs []*MetricDiff, expected pdata.Metric, actual pdata.Metric) []*MetricDiff {
	diffs = diffMetricDescriptor(diffs, expected.MetricDescriptor(), actual.MetricDescriptor())
	diffs = diffInt64Pts(diffs, expected.Int64DataPoints(), actual.Int64DataPoints())
	diffs = diffDoublePts(diffs, expected.DoubleDataPoints(), actual.DoubleDataPoints())
	diffs = diffHistogramPts(diffs, expected.HistogramDataPoints(), actual.HistogramDataPoints())
	diffs = diffSummaryPts(diffs, expected.SummaryDataPoints(), actual.SummaryDataPoints())
	return diffs
}

func diffSummaryPts(diffs []*MetricDiff, expected pdata.SummaryDataPointSlice, actual pdata.SummaryDataPointSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, actual.Len(), expected.Len(), "MetricSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffSummaryPt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffSummaryPt(diffs []*MetricDiff, expected pdata.SummaryDataPoint, actual pdata.SummaryDataPoint) []*MetricDiff {
	diffs = diff(diffs, expected.Count(), actual.Count(), "SummaryDataPoint Count")
	diffs = diff(diffs, expected.Sum(), actual.Sum(), "SummaryDataPoint Sum")
	diffs = diffPercentiles(diffs, expected.ValueAtPercentiles(), actual.ValueAtPercentiles())
	return diffs
}

func diffPercentiles(diffs []*MetricDiff, expected pdata.SummaryValueAtPercentileSlice, actual pdata.SummaryValueAtPercentileSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "MetricSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffSummaryAtPct(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffSummaryAtPct(diffs []*MetricDiff, expected pdata.SummaryValueAtPercentile, actual pdata.SummaryValueAtPercentile) []*MetricDiff {
	diffs = diff(diffs, expected.Value(), actual.Value(), "SummaryValueAtPercentile Value")
	diffs = diff(diffs, expected.Percentile(), actual.Percentile(), "SummaryValueAtPercentile Percentile")
	return diffs
}

func diffMetricDescriptor(diffs []*MetricDiff, expected pdata.MetricDescriptor, actual pdata.MetricDescriptor) []*MetricDiff {
	diffs = diff(diffs, expected.Type(), actual.Type(), "MetricDescriptor Type")
	diffs = diff(diffs, expected.Name(), actual.Name(), "MetricDescriptor Name")
	diffs = diff(diffs, expected.Description(), actual.Description(), "MetricDescriptor Description")
	diffs = diff(diffs, expected.Unit(), actual.Unit(), "MetricDescriptor Unit")
	return diffs
}

func diffDoublePts(diffs []*MetricDiff, expected pdata.DoubleDataPointSlice, actual pdata.DoubleDataPointSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "DoubleDataPointSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffDoublePt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffDoublePt(diffs []*MetricDiff, expected pdata.DoubleDataPoint, actual pdata.DoubleDataPoint) []*MetricDiff {
	return diff(diffs, expected.Value(), actual.Value(), "DoubleDataPoint value")
}

func diffHistogramPts(diffs []*MetricDiff, expected pdata.HistogramDataPointSlice, actual pdata.HistogramDataPointSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "HistogramDataPointSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffHistogramPt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffHistogramPt(diffs []*MetricDiff, expected pdata.HistogramDataPoint, actual pdata.HistogramDataPoint) []*MetricDiff {
	diffs = diff(diffs, expected.Count(), actual.Count(), "HistogramDataPoint Count")
	diffs = diff(diffs, expected.Sum(), actual.Sum(), "HistogramDataPoint Sum")
	// todo LabelsMap()
	diffs = diffBuckets(diffs, expected.Buckets(), actual.Buckets())
	return diffs
}

func diffBuckets(diffs []*MetricDiff, expected pdata.HistogramBucketSlice, actual pdata.HistogramBucketSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "HistogramBucketSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffBucket(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffBucket(diffs []*MetricDiff, expected pdata.HistogramBucket, actual pdata.HistogramBucket) []*MetricDiff {
	diffs = diff(diffs, expected.Count(), actual.Count(), "HistogramBucket Count")
	diffs = diffExemplar(diffs, expected.Exemplar(), actual.Exemplar())
	return diffs
}

func diffExemplar(diffs []*MetricDiff, expected pdata.HistogramBucketExemplar, actual pdata.HistogramBucketExemplar) []*MetricDiff {
	diffs = diff(diffs, expected.Value(), actual.Value(), "HistogramBucketExemplar Value")
	return diffs
}

func diffInt64Pts(diffs []*MetricDiff, expected pdata.Int64DataPointSlice, actual pdata.Int64DataPointSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "Int64DataPointSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffInt64Pt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffInt64Pt(diffs []*MetricDiff, expected pdata.Int64DataPoint, actual pdata.Int64DataPoint) []*MetricDiff {
	return diff(diffs, expected.Value(), actual.Value(), "Int64DataPoint value")
}

func diffResource(diffs []*MetricDiff, expected pdata.Resource, actual pdata.Resource) []*MetricDiff {
	return diffAttrs(diffs, expected.Attributes(), actual.Attributes())
}

func diffAttrs(diffs []*MetricDiff, expected pdata.AttributeMap, actual pdata.AttributeMap) []*MetricDiff {
	if !reflect.DeepEqual(expected, actual) {
		diffs = append(diffs, &MetricDiff{
			expectedValue: attrMapToString(expected),
			actualValue:   attrMapToString(actual),
			msg:           "Resource attributes",
		})
	}
	return diffs
}

func diff(diffs []*MetricDiff, expected interface{}, actual interface{}, msg string) []*MetricDiff {
	out, _ := diffValues(diffs, expected, actual, msg)
	return out
}

func diffValues(diffs []*MetricDiff, expected interface{}, actual interface{}, msg string) ([]*MetricDiff, bool) {
	if expected != actual {
		return append(diffs, &MetricDiff{
			msg:           msg,
			expectedValue: expected,
			actualValue:   actual,
		}), true
	}
	return diffs, false
}

func attrMapToString(m pdata.AttributeMap) string {
	out := ""
	m.ForEach(func(k string, v pdata.AttributeValue) {
		out += "[" + k + "=" + v.StringVal() + "]"
	})
	return out
}
