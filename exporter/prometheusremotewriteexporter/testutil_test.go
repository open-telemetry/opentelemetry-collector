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
	"time"

	"github.com/prometheus/prometheus/prompb"

	commonpb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1old"
	"go.opentelemetry.io/collector/internal/dataold"
)

type combination struct {
	ty   otlp.MetricDescriptor_Type
	temp otlp.MetricDescriptor_Temporality
}

var (
	time1   = uint64(time.Now().UnixNano())
	time2   = uint64(time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).UnixNano())
	msTime1 = int64(time1 / uint64(int64(time.Millisecond)/int64(time.Nanosecond)))
	msTime2 = int64(time2 / uint64(int64(time.Millisecond)/int64(time.Nanosecond)))

	typeInt64           = "INT64"
	typeMonotonicInt64  = "MONOTONIC_INT64"
	typeMonotonicDouble = "MONOTONIC_DOUBLE"
	typeHistogram       = "HISTOGRAM"
	typeSummary         = "SUMMARY"

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
	dirty1  = "%"
	dirty2  = "?"

	intVal1   int64 = 1
	intVal2   int64 = 2
	floatVal1       = 1.0
	floatVal2       = 2.0

	lbs1      = getLabels(label11, value11, label12, value12)
	lbs2      = getLabels(label21, value21, label22, value22)
	lbs1Dirty = getLabels(label11+dirty1, value11, dirty2+label12, value12)

	promLbs1 = getPromLabels(label11, value11, label12, value12)
	promLbs2 = getPromLabels(label21, value21, label22, value22)

	lb1Sig = "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12
	lb2Sig = "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22
	ns1    = "test_ns"
	name1  = "valid_single_int_point"

	monotonicInt64Comb  = 0
	monotonicDoubleComb = 1
	histogramComb       = 2
	summaryComb         = 3
	validCombinations   = []combination{
		{otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_INSTANTANEOUS},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_INSTANTANEOUS},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_CUMULATIVE},
	}
	invalidCombinations = []combination{
		{otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_DELTA},
		{ty: otlp.MetricDescriptor_INVALID_TYPE},
		{temp: otlp.MetricDescriptor_INVALID_TEMPORALITY},
		{},
	}
	twoPointsSameTs = map[string]*prompb.TimeSeries{
		typeInt64 + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1),
			getSample(float64(intVal2), msTime2)),
	}
	twoPointsDifferentTs = map[string]*prompb.TimeSeries{
		typeInt64 + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1)),
		typeInt64 + "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22: getTimeSeries(getPromLabels(label21, value21, label22, value22),
			getSample(float64(intVal1), msTime2)),
	}
)

// OTLP metrics
// labels must come in pairs
func getLabels(labels ...string) []*commonpb.StringKeyValue {
	var set []*commonpb.StringKeyValue
	for i := 0; i < len(labels); i += 2 {
		set = append(set, &commonpb.StringKeyValue{
			Key:   labels[i],
			Value: labels[i+1],
		})
	}
	return set
}

func getDescriptor(name string, i int, comb []combination) *otlp.MetricDescriptor {
	return &otlp.MetricDescriptor{
		Name:        name,
		Description: "",
		Unit:        "",
		Type:        comb[i].ty,
		Temporality: comb[i].temp,
	}
}

func getIntDataPoint(labels []*commonpb.StringKeyValue, value int64, ts uint64) *otlp.Int64DataPoint {
	return &otlp.Int64DataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Value:             value,
	}
}

func getDoubleDataPoint(labels []*commonpb.StringKeyValue, value float64, ts uint64) *otlp.DoubleDataPoint {
	return &otlp.DoubleDataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Value:             value,
	}
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

//setCumulative is for creating the dataold.MetricData to test with
func setTemporality(metricsData *dataold.MetricData, temp otlp.MetricDescriptor_Temporality) {
	for _, r := range dataold.MetricDataToOtlp(*metricsData) {
		for _, instMetrics := range r.InstrumentationLibraryMetrics {
			for _, m := range instMetrics.Metrics {
				m.MetricDescriptor.Temporality = temp
			}
		}
	}
}

//setDataPointToNil is for creating the dataold.MetricData to test with
func setDataPointToNil(metricsData *dataold.MetricData, dataField string) {
	for _, r := range dataold.MetricDataToOtlp(*metricsData) {
		for _, instMetrics := range r.InstrumentationLibraryMetrics {
			for _, m := range instMetrics.Metrics {
				switch dataField {
				case typeMonotonicInt64:
					m.Int64DataPoints = nil
				case typeMonotonicDouble:
					m.DoubleDataPoints = nil
				case typeHistogram:
					m.HistogramDataPoints = nil
				case typeSummary:
					m.SummaryDataPoints = nil
				}
			}
		}
	}
}

//setType is for creating the dataold.MetricData to test with
func setType(metricsData *dataold.MetricData, dataField string) {
	for _, r := range dataold.MetricDataToOtlp(*metricsData) {
		for _, instMetrics := range r.InstrumentationLibraryMetrics {
			for _, m := range instMetrics.Metrics {
				switch dataField {
				case typeMonotonicInt64:
					m.GetMetricDescriptor().Type = otlp.MetricDescriptor_MONOTONIC_INT64
				case typeMonotonicDouble:
					m.GetMetricDescriptor().Type = otlp.MetricDescriptor_MONOTONIC_DOUBLE
				case typeHistogram:
					m.GetMetricDescriptor().Type = otlp.MetricDescriptor_HISTOGRAM
				case typeSummary:
					m.GetMetricDescriptor().Type = otlp.MetricDescriptor_SUMMARY
				}
			}
		}
	}
}
