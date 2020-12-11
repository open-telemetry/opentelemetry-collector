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

	commonpb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
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
		"2" + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1),
			getSample(float64(intVal2), msTime2)),
	}
	twoPointsDifferentTs = map[string]*prompb.TimeSeries{
		"1" + "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12: getTimeSeries(getPromLabels(label11, value11, label12, value12),
			getSample(float64(intVal1), msTime1)),
		"1" + "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22: getTimeSeries(getPromLabels(label21, value21, label22, value22),
			getSample(float64(intVal1), msTime2)),
	}
	bounds  = []float64{0.1, 0.5, 0.99}
	buckets = []uint64{1, 2, 3}

	quantileBounds = []float64{0.15, 0.9, 0.99}
	quantileValues = []float64{7, 8, 9}
	quantiles      = getQuantiles(quantileBounds, quantileValues)

	validIntGauge        = "valid_IntGauge"
	validDoubleGauge     = "valid_DoubleGauge"
	validIntSum          = "valid_IntSum"
	validDoubleSum       = "valid_DoubleSum"
	validIntHistogram    = "valid_IntHistogram"
	validDoubleHistogram = "valid_DoubleHistogram"
	validDoubleSummary   = "valid_DoubleSummary"

	validIntGaugeDirty = "*valid_IntGauge$"

	unmatchedBoundBucketIntHist    = "unmatchedBoundBucketIntHist"
	unmatchedBoundBucketDoubleHist = "unmatchedBoundBucketDoubleHist"

	// valid metrics as input should not return error
	validMetrics1 = map[string]*otlp.Metric{
		validIntGauge: {
			Name: validIntGauge,
			Data: &otlp.Metric_IntGauge{
				IntGauge: &otlp.IntGauge{
					DataPoints: []*otlp.IntDataPoint{
						getIntDataPoint(lbs1, intVal1, time1),
						nil,
					},
				},
			},
		},
		validDoubleGauge: {
			Name: validDoubleGauge,
			Data: &otlp.Metric_DoubleGauge{
				DoubleGauge: &otlp.DoubleGauge{
					DataPoints: []*otlp.DoubleDataPoint{
						getDoubleDataPoint(lbs1, floatVal1, time1),
						nil,
					},
				},
			},
		},
		validIntSum: {
			Name: validIntSum,
			Data: &otlp.Metric_IntSum{
				IntSum: &otlp.IntSum{
					DataPoints: []*otlp.IntDataPoint{
						getIntDataPoint(lbs1, intVal1, time1),
						nil,
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validDoubleSum: {
			Name: validDoubleSum,
			Data: &otlp.Metric_DoubleSum{
				DoubleSum: &otlp.DoubleSum{
					DataPoints: []*otlp.DoubleDataPoint{
						getDoubleDataPoint(lbs1, floatVal1, time1),
						nil,
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validIntHistogram: {
			Name: validIntHistogram,
			Data: &otlp.Metric_IntHistogram{
				IntHistogram: &otlp.IntHistogram{
					DataPoints: []*otlp.IntHistogramDataPoint{
						getIntHistogramDataPoint(lbs1, time1, floatVal1, uint64(intVal1), bounds, buckets),
						nil,
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validDoubleHistogram: {
			Name: validDoubleHistogram,
			Data: &otlp.Metric_DoubleHistogram{
				DoubleHistogram: &otlp.DoubleHistogram{
					DataPoints: []*otlp.DoubleHistogramDataPoint{
						getDoubleHistogramDataPoint(lbs1, time1, floatVal1, uint64(intVal1), bounds, buckets),
						nil,
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validDoubleSummary: {
			Name: validDoubleSummary,
			Data: &otlp.Metric_DoubleSummary{
				DoubleSummary: &otlp.DoubleSummary{
					DataPoints: []*otlp.DoubleSummaryDataPoint{
						getDoubleSummaryDataPoint(lbs1, time1, floatVal1, uint64(intVal1), quantiles),
						nil,
					},
				},
			},
		},
	}
	validMetrics2 = map[string]*otlp.Metric{
		validIntGauge: {
			Name: validIntGauge,
			Data: &otlp.Metric_IntGauge{
				IntGauge: &otlp.IntGauge{
					DataPoints: []*otlp.IntDataPoint{
						getIntDataPoint(lbs2, intVal2, time2),
					},
				},
			},
		},
		validDoubleGauge: {
			Name: validDoubleGauge,
			Data: &otlp.Metric_DoubleGauge{
				DoubleGauge: &otlp.DoubleGauge{
					DataPoints: []*otlp.DoubleDataPoint{
						getDoubleDataPoint(lbs2, floatVal2, time2),
					},
				},
			},
		},
		validIntSum: {
			Name: validIntSum,
			Data: &otlp.Metric_IntSum{
				IntSum: &otlp.IntSum{
					DataPoints: []*otlp.IntDataPoint{
						getIntDataPoint(lbs2, intVal2, time2),
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validDoubleSum: {
			Name: validDoubleSum,
			Data: &otlp.Metric_DoubleSum{
				DoubleSum: &otlp.DoubleSum{
					DataPoints: []*otlp.DoubleDataPoint{
						getDoubleDataPoint(lbs2, floatVal2, time2),
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validIntHistogram: {
			Name: validIntHistogram,
			Data: &otlp.Metric_IntHistogram{
				IntHistogram: &otlp.IntHistogram{
					DataPoints: []*otlp.IntHistogramDataPoint{
						getIntHistogramDataPoint(lbs2, time2, floatVal2, uint64(intVal2), bounds, buckets),
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validDoubleHistogram: {
			Name: validDoubleHistogram,
			Data: &otlp.Metric_DoubleHistogram{
				DoubleHistogram: &otlp.DoubleHistogram{
					DataPoints: []*otlp.DoubleHistogramDataPoint{
						getDoubleHistogramDataPoint(lbs2, time2, floatVal2, uint64(intVal2), bounds, buckets),
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		validDoubleSummary: {
			Name: validDoubleSummary,
			Data: &otlp.Metric_DoubleSummary{
				DoubleSummary: &otlp.DoubleSummary{
					DataPoints: []*otlp.DoubleSummaryDataPoint{
						getDoubleSummaryDataPoint(lbs2, time2, floatVal2, uint64(intVal2), quantiles),
						nil,
					},
				},
			},
		},
		validIntGaugeDirty: {
			Name: validIntGaugeDirty,
			Data: &otlp.Metric_IntGauge{
				IntGauge: &otlp.IntGauge{
					DataPoints: []*otlp.IntDataPoint{
						getIntDataPoint(lbs1, intVal1, time1),
						nil,
					},
				},
			},
		},
		unmatchedBoundBucketIntHist: {
			Name: unmatchedBoundBucketIntHist,
			Data: &otlp.Metric_IntHistogram{
				IntHistogram: &otlp.IntHistogram{
					DataPoints: []*otlp.IntHistogramDataPoint{
						{
							ExplicitBounds: []float64{0.1, 0.2, 0.3},
							BucketCounts:   []uint64{1, 2},
						},
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		unmatchedBoundBucketDoubleHist: {
			Name: unmatchedBoundBucketDoubleHist,
			Data: &otlp.Metric_DoubleHistogram{
				DoubleHistogram: &otlp.DoubleHistogram{
					DataPoints: []*otlp.DoubleHistogramDataPoint{
						{
							ExplicitBounds: []float64{0.1, 0.2, 0.3},
							BucketCounts:   []uint64{1, 2},
						},
					},
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
	}

	nilMetric = "nil"
	empty     = "empty"

	// Category 1: type and data field doesn't match
	notMatchIntGauge        = "noMatchIntGauge"
	notMatchDoubleGauge     = "notMatchDoubleGauge"
	notMatchIntSum          = "notMatchIntSum"
	notMatchDoubleSum       = "notMatchDoubleSum"
	notMatchIntHistogram    = "notMatchIntHistogram"
	notMatchDoubleHistogram = "notMatchDoubleHistogram"
	notMatchDoubleSummary   = "notMatchDoubleSummary"

	// Category 2: invalid type and temporality combination
	invalidIntSum          = "invalidIntSum"
	invalidDoubleSum       = "invalidDoubleSum"
	invalidIntHistogram    = "invalidIntHistogram"
	invalidDoubleHistogram = "invalidDoubleHistogram"

	// Category 3: nil data points
	nilDataPointIntGauge        = "nilDataPointIntGauge"
	nilDataPointDoubleGauge     = "nilDataPointDoubleGauge"
	nilDataPointIntSum          = "nilDataPointIntSum"
	nilDataPointDoubleSum       = "nilDataPointDoubleSum"
	nilDataPointIntHistogram    = "nilDataPointIntHistogram"
	nilDataPointDoubleHistogram = "nilDataPointDoubleHistogram"
	nilDataPointDoubleSummary   = "nilDataPointDoubleSummary"

	// different metrics that will not pass validate metrics
	invalidMetrics = map[string]*otlp.Metric{
		// nil
		nilMetric: nil,
		// Data = nil
		empty: {},
		notMatchIntGauge: {
			Name: notMatchIntGauge,
			Data: &otlp.Metric_IntGauge{},
		},
		notMatchDoubleGauge: {
			Name: notMatchDoubleGauge,
			Data: &otlp.Metric_DoubleGauge{},
		},
		notMatchIntSum: {
			Name: notMatchIntSum,
			Data: &otlp.Metric_IntSum{},
		},
		notMatchDoubleSum: {
			Name: notMatchDoubleSum,
			Data: &otlp.Metric_DoubleSum{},
		},
		notMatchIntHistogram: {
			Name: notMatchIntHistogram,
			Data: &otlp.Metric_IntHistogram{},
		},
		notMatchDoubleHistogram: {
			Name: notMatchDoubleHistogram,
			Data: &otlp.Metric_DoubleHistogram{},
		},
		notMatchDoubleSummary: {
			Name: notMatchDoubleSummary,
			Data: &otlp.Metric_DoubleSummary{},
		},
		invalidIntSum: {
			Name: invalidIntSum,
			Data: &otlp.Metric_IntSum{
				IntSum: &otlp.IntSum{
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				},
			},
		},
		invalidDoubleSum: {
			Name: invalidDoubleSum,
			Data: &otlp.Metric_DoubleSum{
				DoubleSum: &otlp.DoubleSum{
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				},
			},
		},
		invalidIntHistogram: {
			Name: invalidIntHistogram,
			Data: &otlp.Metric_IntHistogram{
				IntHistogram: &otlp.IntHistogram{
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				},
			},
		},
		invalidDoubleHistogram: {
			Name: invalidDoubleHistogram,
			Data: &otlp.Metric_DoubleHistogram{
				DoubleHistogram: &otlp.DoubleHistogram{
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				},
			},
		},
	}

	// different metrics that will cause the exporter to return an error
	errorMetrics = map[string]*otlp.Metric{

		nilDataPointIntGauge: {
			Name: nilDataPointIntGauge,
			Data: &otlp.Metric_IntGauge{
				IntGauge: &otlp.IntGauge{DataPoints: nil},
			},
		},
		nilDataPointDoubleGauge: {
			Name: nilDataPointDoubleGauge,
			Data: &otlp.Metric_DoubleGauge{
				DoubleGauge: &otlp.DoubleGauge{DataPoints: nil},
			},
		},
		nilDataPointIntSum: {
			Name: nilDataPointIntSum,
			Data: &otlp.Metric_IntSum{
				IntSum: &otlp.IntSum{
					DataPoints:             nil,
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		nilDataPointDoubleSum: {
			Name: nilDataPointDoubleSum,
			Data: &otlp.Metric_DoubleSum{
				DoubleSum: &otlp.DoubleSum{
					DataPoints:             nil,
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		nilDataPointIntHistogram: {
			Name: nilDataPointIntHistogram,
			Data: &otlp.Metric_IntHistogram{
				IntHistogram: &otlp.IntHistogram{
					DataPoints:             nil,
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		nilDataPointDoubleHistogram: {
			Name: nilDataPointDoubleHistogram,
			Data: &otlp.Metric_DoubleHistogram{
				DoubleHistogram: &otlp.DoubleHistogram{
					DataPoints:             nil,
					AggregationTemporality: otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
		nilDataPointDoubleSummary: {
			Name: nilDataPointDoubleSummary,
			Data: &otlp.Metric_DoubleSummary{
				DoubleSummary: &otlp.DoubleSummary{
					DataPoints: nil,
				},
			},
		},
	}
)

// OTLP metrics
// labels must come in pairs
func getLabels(labels ...string) []commonpb.StringKeyValue {
	var set []commonpb.StringKeyValue
	for i := 0; i < len(labels); i += 2 {
		set = append(set, commonpb.StringKeyValue{
			Key:   labels[i],
			Value: labels[i+1],
		})
	}
	return set
}

func getIntDataPoint(labels []commonpb.StringKeyValue, value int64, ts uint64) *otlp.IntDataPoint {
	return &otlp.IntDataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Value:             value,
	}
}

func getDoubleDataPoint(labels []commonpb.StringKeyValue, value float64, ts uint64) *otlp.DoubleDataPoint {
	return &otlp.DoubleDataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Value:             value,
	}
}

func getIntHistogramDataPoint(labels []commonpb.StringKeyValue, ts uint64, sum float64, count uint64, bounds []float64,
	buckets []uint64) *otlp.IntHistogramDataPoint {
	return &otlp.IntHistogramDataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Count:             count,
		Sum:               int64(sum),
		BucketCounts:      buckets,
		ExplicitBounds:    bounds,
		Exemplars:         nil,
	}
}

func getDoubleHistogramDataPoint(labels []commonpb.StringKeyValue, ts uint64, sum float64, count uint64,
	bounds []float64, buckets []uint64) *otlp.DoubleHistogramDataPoint {
	return &otlp.DoubleHistogramDataPoint{
		Labels:         labels,
		TimeUnixNano:   ts,
		Count:          count,
		Sum:            sum,
		BucketCounts:   buckets,
		ExplicitBounds: bounds,
	}
}

func getDoubleSummaryDataPoint(labels []commonpb.StringKeyValue, ts uint64, sum float64, count uint64,
	quantiles []*otlp.DoubleSummaryDataPoint_ValueAtQuantile) *otlp.DoubleSummaryDataPoint {
	return &otlp.DoubleSummaryDataPoint{
		Labels:         labels,
		TimeUnixNano:   ts,
		Count:          count,
		Sum:            sum,
		QuantileValues: quantiles,
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

func getQuantiles(bounds []float64, values []float64) []*otlp.DoubleSummaryDataPoint_ValueAtQuantile {
	quantiles := make([]*otlp.DoubleSummaryDataPoint_ValueAtQuantile, len(bounds))
	for i := 0; i < len(bounds); i++ {
		quantiles[i] = &otlp.DoubleSummaryDataPoint_ValueAtQuantile{
			Quantile: bounds[i],
			Value:    values[i],
		}
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
