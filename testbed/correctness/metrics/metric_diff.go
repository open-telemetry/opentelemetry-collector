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

package metrics

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/model/pdata"
)

// MetricDiff is intended to support producing human-readable diffs between two MetricData structs during
// testing. Two MetricDatas, when compared, could produce a list of MetricDiffs containing all of their
// differences, which could be used to correct the differences between the expected and actual values.
type MetricDiff struct {
	ExpectedValue interface{}
	ActualValue   interface{}
	Msg           string
}

func (mf MetricDiff) String() string {
	return fmt.Sprintf("{msg='%v' expected=[%v] actual=[%v]}\n", mf.Msg, mf.ExpectedValue, mf.ActualValue)
}

func diffRMSlices(sent []pdata.ResourceMetrics, recd []pdata.ResourceMetrics) []*MetricDiff {
	var diffs []*MetricDiff
	if len(sent) != len(recd) {
		return []*MetricDiff{{
			ExpectedValue: len(sent),
			ActualValue:   len(recd),
			Msg:           "Sent vs received ResourceMetrics not equal length",
		}}
	}
	for i := 0; i < len(sent); i++ {
		sentRM := sent[i]
		recdRM := recd[i]
		diffs = diffRMs(diffs, sentRM, recdRM)
	}
	return diffs
}

func diffRMs(diffs []*MetricDiff, expected pdata.ResourceMetrics, actual pdata.ResourceMetrics) []*MetricDiff {
	diffs = diffResource(diffs, expected.Resource(), actual.Resource())
	diffs = diffILMSlice(
		diffs,
		expected.InstrumentationLibraryMetrics(),
		actual.InstrumentationLibraryMetrics(),
	)
	return diffs
}

func diffILMSlice(
	diffs []*MetricDiff,
	expected pdata.InstrumentationLibraryMetricsSlice,
	actual pdata.InstrumentationLibraryMetricsSlice,
) []*MetricDiff {
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

func diffILM(
	diffs []*MetricDiff,
	expected pdata.InstrumentationLibraryMetrics,
	actual pdata.InstrumentationLibraryMetrics,
) []*MetricDiff {
	return diffMetrics(diffs, expected.Metrics(), actual.Metrics())
}

func diffMetrics(diffs []*MetricDiff, expected pdata.MetricSlice, actual pdata.MetricSlice) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, actual.Len(), expected.Len(), "MetricSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = DiffMetric(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func DiffMetric(diffs []*MetricDiff, expected pdata.Metric, actual pdata.Metric) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffMetricDescriptor(diffs, expected, actual)
	if mismatch {
		return diffs
	}
	switch actual.DataType() {
	case pdata.MetricDataTypeIntGauge:
		diffs = diffIntPts(diffs, expected.IntGauge().DataPoints(), actual.IntGauge().DataPoints())
	case pdata.MetricDataTypeGauge:
		diffs = diffDoublePts(diffs, expected.Gauge().DataPoints(), actual.Gauge().DataPoints())
	case pdata.MetricDataTypeIntSum:
		diffs = diff(diffs, expected.IntSum().IsMonotonic(), actual.IntSum().IsMonotonic(), "IntSum IsMonotonic")
		diffs = diff(diffs, expected.IntSum().AggregationTemporality(), actual.IntSum().AggregationTemporality(), "IntSum AggregationTemporality")
		diffs = diffIntPts(diffs, expected.IntSum().DataPoints(), actual.IntSum().DataPoints())
	case pdata.MetricDataTypeSum:
		diffs = diff(diffs, expected.Sum().IsMonotonic(), actual.Sum().IsMonotonic(), "Sum IsMonotonic")
		diffs = diff(diffs, expected.Sum().AggregationTemporality(), actual.Sum().AggregationTemporality(), "Sum AggregationTemporality")
		diffs = diffDoublePts(diffs, expected.Sum().DataPoints(), actual.Sum().DataPoints())
	case pdata.MetricDataTypeIntHistogram:
		diffs = diff(diffs, expected.IntHistogram().AggregationTemporality(), actual.IntHistogram().AggregationTemporality(), "IntHistogram AggregationTemporality")
		diffs = diffIntHistogramPts(diffs, expected.IntHistogram().DataPoints(), actual.IntHistogram().DataPoints())
	case pdata.MetricDataTypeHistogram:
		diffs = diff(diffs, expected.Histogram().AggregationTemporality(), actual.Histogram().AggregationTemporality(), "Histogram AggregationTemporality")
		diffs = diffDoubleHistogramPts(diffs, expected.Histogram().DataPoints(), actual.Histogram().DataPoints())
	}
	return diffs
}

func diffMetricDescriptor(
	diffs []*MetricDiff,
	expected pdata.Metric,
	actual pdata.Metric,
) ([]*MetricDiff, bool) {
	diffs = diff(diffs, expected.Name(), actual.Name(), "Metric Name")
	diffs = diff(diffs, expected.Description(), actual.Description(), "Metric Description")
	diffs = diff(diffs, expected.Unit(), actual.Unit(), "Metric Unit")
	return diffValues(diffs, expected.DataType(), actual.DataType(), "Metric Type")
}

func diffDoublePts(
	diffs []*MetricDiff,
	expected pdata.DoubleDataPointSlice,
	actual pdata.DoubleDataPointSlice,
) []*MetricDiff {
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

func diffDoublePt(
	diffs []*MetricDiff,
	expected pdata.DoubleDataPoint,
	actual pdata.DoubleDataPoint,
) []*MetricDiff {
	diffs = diff(diffs, expected.Value(), actual.Value(), "DoubleDataPoint value")
	return diffDoubleExemplars(diffs, expected.Exemplars(), actual.Exemplars())
}

func diffDoubleHistogramPts(
	diffs []*MetricDiff,
	expected pdata.HistogramDataPointSlice,
	actual pdata.HistogramDataPointSlice,
) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "HistogramDataPointSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffDoubleHistogramPt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffDoubleHistogramPt(
	diffs []*MetricDiff,
	expected pdata.HistogramDataPoint,
	actual pdata.HistogramDataPoint,
) []*MetricDiff {
	diffs = diff(diffs, expected.Count(), actual.Count(), "HistogramDataPoint Count")
	diffs = diff(diffs, expected.Sum(), actual.Sum(), "HistogramDataPoint Sum")
	diffs = diff(diffs, expected.BucketCounts(), actual.BucketCounts(), "HistogramDataPoint BucketCounts")
	diffs = diff(diffs, expected.ExplicitBounds(), actual.ExplicitBounds(), "HistogramDataPoint ExplicitBounds")
	// todo LabelsMap()
	return diffDoubleExemplars(diffs, expected.Exemplars(), actual.Exemplars())
}

func diffDoubleExemplars(
	diffs []*MetricDiff,
	expected pdata.ExemplarSlice,
	actual pdata.ExemplarSlice,
) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "ExemplarSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diff(diffs, expected.At(i).Value(), actual.At(i).Value(), "Exemplar Value")
	}
	return diffs
}

func diffIntHistogramPts(
	diffs []*MetricDiff,
	expected pdata.IntHistogramDataPointSlice,
	actual pdata.IntHistogramDataPointSlice,
) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "HistogramDataPointSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffIntHistogramPt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffIntHistogramPt(
	diffs []*MetricDiff,
	expected pdata.IntHistogramDataPoint,
	actual pdata.IntHistogramDataPoint,
) []*MetricDiff {
	diffs = diff(diffs, expected.Count(), actual.Count(), "HistogramDataPoint Count")
	diffs = diff(diffs, expected.Sum(), actual.Sum(), "HistogramDataPoint Sum")
	diffs = diff(diffs, expected.BucketCounts(), actual.BucketCounts(), "HistogramDataPoint BucketCounts")
	diffs = diff(diffs, expected.ExplicitBounds(), actual.ExplicitBounds(), "HistogramDataPoint ExplicitBounds")
	// todo LabelsMap()
	return diffIntExemplars(diffs, expected.Exemplars(), actual.Exemplars())
}

func diffIntExemplars(
	diffs []*MetricDiff,
	expected pdata.IntExemplarSlice,
	actual pdata.IntExemplarSlice,
) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "ExemplarSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diff(diffs, expected.At(i).Value(), actual.At(i).Value(), "Exemplar Value")
	}
	return diffs
}

func diffIntPts(
	diffs []*MetricDiff,
	expected pdata.IntDataPointSlice,
	actual pdata.IntDataPointSlice,
) []*MetricDiff {
	var mismatch bool
	diffs, mismatch = diffValues(diffs, expected.Len(), actual.Len(), "IntDataPointSlice len")
	if mismatch {
		return diffs
	}
	for i := 0; i < expected.Len(); i++ {
		diffs = diffIntPt(diffs, expected.At(i), actual.At(i))
	}
	return diffs
}

func diffIntPt(
	diffs []*MetricDiff,
	expected pdata.IntDataPoint,
	actual pdata.IntDataPoint,
) []*MetricDiff {
	return diff(diffs, expected.Value(), actual.Value(), "IntDataPoint value")
}

func diffResource(diffs []*MetricDiff, expected pdata.Resource, actual pdata.Resource) []*MetricDiff {
	return diffAttrs(diffs, expected.Attributes(), actual.Attributes())
}

func diffAttrs(diffs []*MetricDiff, expected pdata.AttributeMap, actual pdata.AttributeMap) []*MetricDiff {
	if !reflect.DeepEqual(expected, actual) {
		diffs = append(diffs, &MetricDiff{
			ExpectedValue: attrMapToString(expected),
			ActualValue:   attrMapToString(actual),
			Msg:           "Resource attributes",
		})
	}
	return diffs
}

func diff(diffs []*MetricDiff, expected interface{}, actual interface{}, msg string) []*MetricDiff {
	out, _ := diffValues(diffs, expected, actual, msg)
	return out
}

func diffValues(
	diffs []*MetricDiff,
	expected interface{},
	actual interface{},
	msg string,
) ([]*MetricDiff, bool) {
	if !reflect.DeepEqual(expected, actual) {
		return append(diffs, &MetricDiff{
			Msg:           msg,
			ExpectedValue: expected,
			ActualValue:   actual,
		}), true
	}
	return diffs, false
}

func attrMapToString(m pdata.AttributeMap) string {
	out := ""
	m.Range(func(k string, v pdata.AttributeValue) bool {
		out += "[" + k + "=" + v.StringVal() + "]"
		return true
	})
	return out
}
