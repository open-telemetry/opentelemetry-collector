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

package prometheusremotewriteexporter

import (
	"fmt"
	"time"

	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/model/pdata"
)

var (
	time1   = uint64(time.Now().UnixNano())
	time2   = uint64(time.Now().UnixNano() - 5)
	time3   = uint64(time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).UnixNano())
	msTime1 = int64(time1 / uint64(int64(time.Millisecond)/int64(time.Nanosecond)))
	msTime2 = int64(time2 / uint64(int64(time.Millisecond)/int64(time.Nanosecond)))
	msTime3 = int64(time3 / uint64(int64(time.Millisecond)/int64(time.Nanosecond)))

	label11 = "test_label11"
	value11 = "test_value11"
	label12 = "test_label12"
	value12 = "test_value12"
	label21 = "test_label21"
	value21 = "test_value21"
	label22 = "test_label22"
	value22 = "test_value22"
	label31 = "test_label31"
	value31 = "test_value31"
	label32 = "test_label32"
	value32 = "test_value32"
	label41 = "__test_label41__"
	value41 = "test_value41"
	dirty1  = "%"
	dirty2  = "?"

	intVal1   int64 = 1
	intVal2   int64 = 2
	floatVal1       = 1.0
	floatVal2       = 2.0
	floatVal3       = 3.0

	lbs1      = getLabels(label11, value11, label12, value12)
	lbs2      = getLabels(label21, value21, label22, value22)
	lbs1Dirty = getLabels(label11+dirty1, value11, dirty2+label12, value12)

	exlbs1 = map[string]string{label41: value41}
	exlbs2 = map[string]string{label11: value41}

	promLbs1 = getPromLabels(label11, value11, label12, value12)
	promLbs2 = getPromLabels(label21, value21, label22, value22)

	lb1Sig = "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12
	lb2Sig = "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22
	ns1    = "test_ns"

	twoPointsSameTs = map[string]*prompb.TimeSeries{
		"DoubleGauge" + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1),
			getSample(float64(intVal2), msTime2)),
	}
	twoPointsDifferentTs = map[string]*prompb.TimeSeries{
		"IntGauge" + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1)),
		"IntGauge" + "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22: getTimeSeries(getPromLabels(label21, value21, label22, value22),
			getSample(float64(intVal1), msTime2)),
	}
	bounds  = []float64{0.1, 0.5, 0.99}
	buckets = []uint64{1, 2, 3}

	quantileBounds = []float64{0.15, 0.9, 0.99}
	quantileValues = []float64{7, 8, 9}
	quantiles      = getQuantiles(quantileBounds, quantileValues)

	validIntGauge     = "valid_IntGauge"
	validDoubleGauge  = "valid_DoubleGauge"
	validIntSum       = "valid_IntSum"
	validDoubleSum    = "valid_DoubleSum"
	validIntHistogram = "valid_IntHistogram"
	validHistogram    = "valid_Histogram"
	validSummary      = "valid_Summary"
	suffixedCounter   = "valid_IntSum_total"

	validIntGaugeDirty = "*valid_IntGauge$"

	unmatchedBoundBucketIntHist = "unmatchedBoundBucketIntHist"
	unmatchedBoundBucketHist    = "unmatchedBoundBucketHist"

	// valid metrics as input should not return error
	validMetrics1 = map[string]pdata.Metric{
		validIntGauge:     getIntGaugeMetric(validIntGauge, lbs1, intVal1, time1),
		validDoubleGauge:  getDoubleGaugeMetric(validDoubleGauge, lbs1, floatVal1, time1),
		validIntSum:       getIntSumMetric(validIntSum, lbs1, intVal1, time1),
		suffixedCounter:   getIntSumMetric(suffixedCounter, lbs1, intVal1, time1),
		validDoubleSum:    getDoubleSumMetric(validDoubleSum, lbs1, floatVal1, time1),
		validIntHistogram: getIntHistogramMetric(validIntHistogram, lbs1, time1, floatVal1, uint64(intVal1), bounds, buckets),
		validHistogram:    getHistogramMetric(validHistogram, lbs1, time1, floatVal1, uint64(intVal1), bounds, buckets),
		validSummary:      getSummaryMetric(validSummary, lbs1, time1, floatVal1, uint64(intVal1), quantiles),
	}
	validMetrics2 = map[string]pdata.Metric{
		validIntGauge:               getIntGaugeMetric(validIntGauge, lbs2, intVal2, time2),
		validDoubleGauge:            getDoubleGaugeMetric(validDoubleGauge, lbs2, floatVal2, time2),
		validIntSum:                 getIntSumMetric(validIntSum, lbs2, intVal2, time2),
		validDoubleSum:              getDoubleSumMetric(validDoubleSum, lbs2, floatVal2, time2),
		validIntHistogram:           getIntHistogramMetric(validIntHistogram, lbs2, time2, floatVal2, uint64(intVal2), bounds, buckets),
		validHistogram:              getHistogramMetric(validHistogram, lbs2, time2, floatVal2, uint64(intVal2), bounds, buckets),
		validSummary:                getSummaryMetric(validSummary, lbs2, time2, floatVal2, uint64(intVal2), quantiles),
		validIntGaugeDirty:          getIntGaugeMetric(validIntGaugeDirty, lbs1, intVal1, time1),
		unmatchedBoundBucketIntHist: getIntHistogramMetric(unmatchedBoundBucketIntHist, pdata.NewStringMap(), 0, 0, 0, []float64{0.1, 0.2, 0.3}, []uint64{1, 2}),
		unmatchedBoundBucketHist:    getHistogramMetric(unmatchedBoundBucketHist, pdata.NewStringMap(), 0, 0, 0, []float64{0.1, 0.2, 0.3}, []uint64{1, 2}),
	}

	empty = "empty"

	// Category 1: type and data field doesn't match
	emptyIntGauge     = "emptyIntGauge"
	emptyDoubleGauge  = "emptyDoubleGauge"
	emptyIntSum       = "emptyIntSum"
	emptyDoubleSum    = "emptyDoubleSum"
	emptyIntHistogram = "emptyIntHistogram"
	emptyHistogram    = "emptyHistogram"
	emptySummary      = "emptySummary"

	// Category 2: invalid type and temporality combination
	emptyCumulativeIntSum       = "emptyCumulativeIntSum"
	emptyCumulativeDoubleSum    = "emptyCumulativeDoubleSum"
	emptyCumulativeIntHistogram = "emptyCumulativeIntHistogram"
	emptyCumulativeHistogram    = "emptyCumulativeHistogram"

	// different metrics that will not pass validate metrics and will cause the exporter to return an error
	invalidMetrics = map[string]pdata.Metric{
		empty:                       pdata.NewMetric(),
		emptyIntGauge:               getEmptyIntGaugeMetric(emptyIntGauge),
		emptyDoubleGauge:            getEmptyDoubleGaugeMetric(emptyDoubleGauge),
		emptyIntSum:                 getEmptyIntSumMetric(emptyIntSum),
		emptyDoubleSum:              getEmptyDoubleSumMetric(emptyDoubleSum),
		emptyIntHistogram:           getEmptyIntHistogramMetric(emptyIntHistogram),
		emptyHistogram:              getEmptyHistogramMetric(emptyHistogram),
		emptySummary:                getEmptySummaryMetric(emptySummary),
		emptyCumulativeIntSum:       getEmptyCumulativeIntSumMetric(emptyCumulativeIntSum),
		emptyCumulativeDoubleSum:    getEmptyCumulativeDoubleSumMetric(emptyCumulativeDoubleSum),
		emptyCumulativeIntHistogram: getEmptyCumulativeIntHistogramMetric(emptyCumulativeIntHistogram),
		emptyCumulativeHistogram:    getEmptyCumulativeHistogramMetric(emptyCumulativeHistogram),
	}
)

// OTLP metrics
// labels must come in pairs
func getLabels(labels ...string) pdata.StringMap {
	stringMap := pdata.NewStringMap()
	for i := 0; i < len(labels); i += 2 {
		stringMap.Upsert(labels[i], labels[i+1])
	}
	return stringMap
}

// Prometheus TimeSeries
func getPromLabels(lbs ...string) []prompb.Label {
	pbLbs := prompb.Labels{
		Labels: []prompb.Label{},
	}
	for i := 0; i < len(lbs); i += 2 {
		pbLbs.Labels = append(pbLbs.Labels, getLabel(lbs[i], lbs[i+1]))
	}
	return pbLbs.Labels
}

func getLabel(name string, value string) prompb.Label {
	return prompb.Label{
		Name:  name,
		Value: value,
	}
}

func getSample(v float64, t int64) prompb.Sample {
	return prompb.Sample{
		Value:     v,
		Timestamp: t,
	}
}

func getTimeSeries(labels []prompb.Label, samples ...prompb.Sample) *prompb.TimeSeries {
	return &prompb.TimeSeries{
		Labels:  labels,
		Samples: samples,
	}
}

func getQuantiles(bounds []float64, values []float64) pdata.ValueAtQuantileSlice {
	quantiles := pdata.NewValueAtQuantileSlice()
	quantiles.Resize(len(bounds))

	for i := 0; i < len(bounds); i++ {
		quantile := quantiles.At(i)
		quantile.SetQuantile(bounds[i])
		quantile.SetValue(values[i])
	}

	return quantiles
}

func getTimeseriesMap(timeseries []*prompb.TimeSeries) map[string]*prompb.TimeSeries {
	tsMap := make(map[string]*prompb.TimeSeries)
	for i, v := range timeseries {
		tsMap[fmt.Sprintf("%s%d", "timeseries_name", i)] = v
	}
	return tsMap
}

func getEmptyIntGaugeMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntGauge)
	return metric
}

func getIntGaugeMetric(name string, labels pdata.StringMap, value int64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntGauge)
	dp := metric.IntGauge().DataPoints().AppendEmpty()
	dp.SetValue(value)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptyDoubleGaugeMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
	return metric
}

func getDoubleGaugeMetric(name string, labels pdata.StringMap, value float64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
	dp := metric.DoubleGauge().DataPoints().AppendEmpty()
	dp.SetValue(value)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptyIntSumMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	return metric
}

func getEmptyCumulativeIntSumMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}

func getIntSumMetric(name string, labels pdata.StringMap, value int64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.IntSum().DataPoints().AppendEmpty()
	dp.SetValue(value)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptyDoubleSumMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	return metric
}

func getEmptyCumulativeDoubleSumMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}

func getDoubleSumMetric(name string, labels pdata.StringMap, value float64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.DoubleSum().DataPoints().AppendEmpty()
	dp.SetValue(value)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptyIntHistogramMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntHistogram)
	return metric
}

func getEmptyCumulativeIntHistogramMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntHistogram)
	metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}

func getIntHistogramMetric(name string, labels pdata.StringMap, ts uint64, sum float64, count uint64, bounds []float64, buckets []uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeIntHistogram)
	metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.IntHistogram().DataPoints().AppendEmpty()
	dp.SetCount(count)
	dp.SetSum(int64(sum))
	dp.SetBucketCounts(buckets)
	dp.SetExplicitBounds(bounds)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptyHistogramMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeHistogram)
	return metric
}

func getEmptyCumulativeHistogramMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeHistogram)
	metric.Histogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}

func getHistogramMetric(name string, labels pdata.StringMap, ts uint64, sum float64, count uint64, bounds []float64, buckets []uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeHistogram)
	metric.Histogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.Histogram().DataPoints().AppendEmpty()
	dp.SetCount(count)
	dp.SetSum(sum)
	dp.SetBucketCounts(buckets)
	dp.SetExplicitBounds(bounds)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptySummaryMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSummary)
	return metric
}

func getSummaryMetric(name string, labels pdata.StringMap, ts uint64, sum float64, count uint64, quantiles pdata.ValueAtQuantileSlice) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSummary)
	dp := metric.Summary().DataPoints().AppendEmpty()
	dp.SetCount(count)
	dp.SetSum(sum)

	labels.Range(func(k string, v string) bool {
		dp.LabelsMap().Upsert(k, v)
		return true
	})

	dp.SetTimestamp(pdata.Timestamp(ts))

	quantiles.CopyTo(dp.QuantileValues())
	quantiles.At(0).Quantile()

	return metric
}

func getResource(resources ...string) pdata.Resource {
	resource := pdata.NewResource()

	for i := 0; i < len(resources); i += 2 {
		resource.Attributes().Upsert(resources[i], pdata.NewAttributeValueString(resources[i+1]))
	}

	return resource
}

func getMetricsFromMetricList(metricList ...pdata.Metric) pdata.Metrics {
	metrics := pdata.NewMetrics()

	rm := metrics.ResourceMetrics().AppendEmpty()
	ilm := rm.InstrumentationLibraryMetrics().AppendEmpty()
	ilm.Metrics().Resize(len(metricList))
	for i := 0; i < len(metricList); i++ {
		metricList[i].CopyTo(ilm.Metrics().At(i))
	}

	return metrics
}
