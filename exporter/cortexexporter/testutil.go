package cortexexporter

import (
	"github.com/prometheus/prometheus/prompb"
	commonpb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"time"
)
type combination struct {
	ty   otlp.MetricDescriptor_Type
	temp otlp.MetricDescriptor_Temporality
}

var (
	time1 = time.Now()
	time2 = time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)

	typeInt64 = "1"
	typeDouble = "2"
	typeHistogram = "3"

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
	dirty1 = "%"
	dirty2 = "?"

	lbs1 = getLabels(label11, value11, label12, value12)
	lbs2 = getLabels(label21, value21, label22, value22)
	lbs1Dirty = getLabels(label11+dirty1, dirty1+value11, dirty2+label12, dirty2+value12)
	lbs2Dirty = getLabels(label21+dirty1, dirty1+value21, dirty2+label22, dirty2+value22)

	promlbs1 = getPromLabels(label11, value11, label12, value12)
	promlbs2 = getPromLabels(label21, value21, label22, value22)
	promlbs3 = getPromLabels(label31, value31, label32, value32)
	int_val1 int64 = 1
	int_val2 int64 = 2
    float_val1 = 1.0
    float_val2 = 2.0

	int64Cumulative = 9

	validCombinations = []combination{
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
)

// OTLP metrics
// labels must come in pairs
func getLabels (labels...string) []*commonpb.StringKeyValue{
	var set []*commonpb.StringKeyValue
	for i := 0; i < len(labels); i += 2 {
		set = append(set, &commonpb.StringKeyValue{
			labels[i],
			labels[i+1],
		})
	}
	return set
}

func getDescriptor(name string, i int, comb []combination) *otlp.MetricDescriptor {
	return &otlp.MetricDescriptor{
		Name:        name,
		Description: "",
		Unit:        "1",
		Type:        comb[i].ty,
		Temporality: comb[i].temp,
	}
}

func getIntDataPoint(lbls []*commonpb.StringKeyValue, value int64, ts time.Time) *otlp.Int64DataPoint{
	return &otlp.Int64DataPoint{
		Labels:            lbls,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(ts.Unix()),
		Value:             value,
	}
}

func getDoubleDataPoint(lbls []*commonpb.StringKeyValue, value float64, ts time.Time) *otlp.DoubleDataPoint {
	return &otlp.DoubleDataPoint{
		Labels:            lbls,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(ts.Unix()),
		Value:             value,
	}
}

func getHistogramDataPoint(lbls []*commonpb.StringKeyValue, ts time.Time, sum float64, count uint64, bounds []float64, buckets []uint64) *otlp.HistogramDataPoint {
	bks := []*otlp.HistogramDataPoint_Bucket{}
	for _, c := range buckets {
		bks = append(bks, &otlp.HistogramDataPoint_Bucket{
			Count:    c,
			Exemplar: nil,
		})
	}
	return &otlp.HistogramDataPoint{
		Labels:            lbls,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(ts.Unix()),
		Count:             count,
		Sum:               sum,
		Buckets:           bks,
		ExplicitBounds:    bounds,
	}
}

func getSummaryDataPoint(lbls []*commonpb.StringKeyValue, ts time.Time, sum float64, count uint64, pcts []float64, values []float64) *otlp.SummaryDataPoint {
	pcs := []*otlp.SummaryDataPoint_ValueAtPercentile{}
	for i, v := range values {
		pcs = append(pcs, &otlp.SummaryDataPoint_ValueAtPercentile{
			Percentile: pcts[i],
			Value: v,
			})
	}
	return &otlp.SummaryDataPoint{
		Labels:            lbls,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(ts.Unix()),
		Count:             count,
		Sum:               sum,
		PercentileValues:  pcs,
	}
}

// Prometheus TimeSeries
func getPromLabels(lbs ...string) prompb.Labels{
	pbLbs := prompb.Labels{
		Labels:               []prompb.Label{},
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	for i := 0; i < len(lbs); i++ {
		pbLbs.Labels = append(pbLbs.Labels, getLabel(lbs[1],lbs[2]))
	}
	return pbLbs
}

func getLabel(name string, value string) prompb.Label{
	return prompb.Label{
		Name:                 name,
		Value:                value,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
}


func getSample(v float64, t time.Time) prompb.Sample {
	return prompb.Sample{
		Value:                v,
		Timestamp:            t.Unix(),
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
}

func getTimeSeries (lbls prompb.Labels, samples...prompb.Sample) *prompb.TimeSeries{
	return &prompb.TimeSeries{
		Labels:               lbls.Labels,
		Samples:              samples,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
}