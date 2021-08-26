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

	lbs1      = getAttributes(label11, value11, label12, value12)
	lbs2      = getAttributes(label21, value21, label22, value22)
	lbs1Dirty = getAttributes(label11+dirty1, value11, dirty2+label12, value12)

	exlbs1 = map[string]string{label41: value41}
	exlbs2 = map[string]string{label11: value41}

	promLbs1 = getPromLabels(label11, value11, label12, value12)
	promLbs2 = getPromLabels(label21, value21, label22, value22)

	lb1Sig = "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12
	lb2Sig = "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22
	ns1    = "test_ns"

	twoPointsSameTs = map[string]*prompb.TimeSeries{
		"Gauge" + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1),
			getSample(float64(intVal2), msTime2)),
	}
	twoPointsDifferentTs = map[string]*prompb.TimeSeries{
		"Gauge" + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1)),
		"Gauge" + "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22: getTimeSeries(getPromLabels(label21, value21, label22, value22),
			getSample(float64(intVal1), msTime2)),
	}
	bounds  = []float64{0.1, 0.5, 0.99}
	buckets = []uint64{1, 2, 3}

	quantileBounds = []float64{0.15, 0.9, 0.99}
	quantileValues = []float64{7, 8, 9}
	quantiles      = getQuantiles(quantileBounds, quantileValues)

	validIntGauge    = "valid_IntGauge"
	validDoubleGauge = "valid_DoubleGauge"
	validIntSum      = "valid_IntSum"
	validSum         = "valid_Sum"
	validHistogram   = "valid_Histogram"
	validSummary     = "valid_Summary"
	suffixedCounter  = "valid_IntSum_total"

	validIntGaugeDirty = "*valid_IntGauge$"

	unmatchedBoundBucketHist = "unmatchedBoundBucketHist"

	// valid metrics as input should not return error
	validMetrics1 = map[string]pdata.Metric{
		validIntGauge:    getIntGaugeMetric(validIntGauge, lbs1, intVal1, time1),
		validDoubleGauge: getDoubleGaugeMetric(validDoubleGauge, lbs1, floatVal1, time1),
		validIntSum:      getIntSumMetric(validIntSum, lbs1, intVal1, time1),
		suffixedCounter:  getIntSumMetric(suffixedCounter, lbs1, intVal1, time1),
		validSum:         getSumMetric(validSum, lbs1, floatVal1, time1),
		validHistogram:   getHistogramMetric(validHistogram, lbs1, time1, floatVal1, uint64(intVal1), bounds, buckets),
		validSummary:     getSummaryMetric(validSummary, lbs1, time1, floatVal1, uint64(intVal1), quantiles),
	}
	validMetrics2 = map[string]pdata.Metric{
		validIntGauge:            getIntGaugeMetric(validIntGauge, lbs2, intVal2, time2),
		validDoubleGauge:         getDoubleGaugeMetric(validDoubleGauge, lbs2, floatVal2, time2),
		validIntSum:              getIntSumMetric(validIntSum, lbs2, intVal2, time2),
		validSum:                 getSumMetric(validSum, lbs2, floatVal2, time2),
		validHistogram:           getHistogramMetric(validHistogram, lbs2, time2, floatVal2, uint64(intVal2), bounds, buckets),
		validSummary:             getSummaryMetric(validSummary, lbs2, time2, floatVal2, uint64(intVal2), quantiles),
		validIntGaugeDirty:       getIntGaugeMetric(validIntGaugeDirty, lbs1, intVal1, time1),
		unmatchedBoundBucketHist: getHistogramMetric(unmatchedBoundBucketHist, pdata.NewAttributeMap(), 0, 0, 0, []float64{0.1, 0.2, 0.3}, []uint64{1, 2}),
	}

	empty = "empty"

	// Category 1: type and data field doesn't match
	emptyGauge     = "emptyGauge"
	emptySum       = "emptySum"
	emptyHistogram = "emptyHistogram"
	emptySummary   = "emptySummary"

	// Category 2: invalid type and temporality combination
	emptyCumulativeSum       = "emptyCumulativeSum"
	emptyCumulativeHistogram = "emptyCumulativeHistogram"

	// different metrics that will not pass validate metrics and will cause the exporter to return an error
	invalidMetrics = map[string]pdata.Metric{
		empty:                    pdata.NewMetric(),
		emptyGauge:               getEmptyGaugeMetric(emptyGauge),
		emptySum:                 getEmptySumMetric(emptySum),
		emptyHistogram:           getEmptyHistogramMetric(emptyHistogram),
		emptySummary:             getEmptySummaryMetric(emptySummary),
		emptyCumulativeSum:       getEmptyCumulativeSumMetric(emptyCumulativeSum),
		emptyCumulativeHistogram: getEmptyCumulativeHistogramMetric(emptyCumulativeHistogram),
	}
)

// OTLP metrics
// attributes must come in pairs
func getAttributes(labels ...string) pdata.AttributeMap {
	attributeMap := pdata.NewAttributeMap()
	for i := 0; i < len(labels); i += 2 {
		attributeMap.UpsertString(labels[i], labels[i+1])
	}
	return attributeMap
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
	quantiles.EnsureCapacity(len(bounds))

	for i := 0; i < len(bounds); i++ {
		quantile := quantiles.AppendEmpty()
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

func getEmptyGaugeMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeGauge)
	return metric
}

func getIntGaugeMetric(name string, attributes pdata.AttributeMap, value int64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeGauge)
	dp := metric.Gauge().DataPoints().AppendEmpty()
	dp.SetIntVal(value)
	attributes.CopyTo(dp.Attributes())

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getDoubleGaugeMetric(name string, attributes pdata.AttributeMap, value float64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeGauge)
	dp := metric.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleVal(value)
	attributes.CopyTo(dp.Attributes())

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptySumMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSum)
	return metric
}

func getIntSumMetric(name string, attributes pdata.AttributeMap, value int64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSum)
	metric.Sum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.Sum().DataPoints().AppendEmpty()
	dp.SetIntVal(value)
	attributes.CopyTo(dp.Attributes())

	dp.SetStartTimestamp(pdata.Timestamp(0))
	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptyCumulativeSumMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSum)
	metric.Sum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}

func getSumMetric(name string, attributes pdata.AttributeMap, value float64, ts uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSum)
	metric.Sum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.Sum().DataPoints().AppendEmpty()
	dp.SetDoubleVal(value)
	attributes.CopyTo(dp.Attributes())

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

func getHistogramMetric(name string, attributes pdata.AttributeMap, ts uint64, sum float64, count uint64, bounds []float64, buckets []uint64) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeHistogram)
	metric.Histogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	dp := metric.Histogram().DataPoints().AppendEmpty()
	dp.SetCount(count)
	dp.SetSum(sum)
	dp.SetBucketCounts(buckets)
	dp.SetExplicitBounds(bounds)
	attributes.CopyTo(dp.Attributes())

	dp.SetTimestamp(pdata.Timestamp(ts))
	return metric
}

func getEmptySummaryMetric(name string) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSummary)
	return metric
}

func getSummaryMetric(name string, attributes pdata.AttributeMap, ts uint64, sum float64, count uint64, quantiles pdata.ValueAtQuantileSlice) pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeSummary)
	dp := metric.Summary().DataPoints().AppendEmpty()
	dp.SetCount(count)
	dp.SetSum(sum)

	attributes.Range(func(k string, v pdata.AttributeValue) bool {
		dp.Attributes().Upsert(k, v)
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
	ilm.Metrics().EnsureCapacity(len(metricList))
	for i := 0; i < len(metricList); i++ {
		metricList[i].CopyTo(ilm.Metrics().AppendEmpty())
	}

	return metrics
}
