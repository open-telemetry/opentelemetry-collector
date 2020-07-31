package cortexexporter

import (
	commonpb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"time"
)
type combination struct {
	ty   otlp.MetricDescriptor_Type
	temp otlp.MetricDescriptor_Temporality
}
var (
	testLabelName1 = "testName1"
	testLabelValue1 = "testValue1"
	testLabelName2 = "testName2"
	testLabelValue2 = "testValue2"
	testLabelNameDirty1 = "test.Name1"
	testLabelValue1Dirty2 = "test.Value1"
	testLabelNameDirty2 = "test.Name2"
	testLabelValueDirty2 = "test.Value2"

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

// labels must come in pairs
func getLabelSet (labels...string) []*commonpb.StringKeyValue{
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

func getIntDataPoint(lbls []*commonpb.StringKeyValue, value int64,) *otlp.Int64DataPoint{
	return &otlp.Int64DataPoint{
		Labels:            lbls,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(time.Now().Unix()),
		Value:             value,
	}
}

func getDoubleDataPoint(lbls []*commonpb.StringKeyValue, value float64,) *otlp.DoubleDataPoint {
	return &otlp.DoubleDataPoint{
		Labels:            lbls,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(time.Now().Unix()),
		Value:             value,
	}
}

func getHistogramDataPoint(lbls []*commonpb.StringKeyValue, sum float64, count uint64, bounds []float64, buckets []uint64) *otlp.HistogramDataPoint {
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
		TimeUnixNano:      uint64(time.Now().Unix()),
		Count:             count,
		Sum:               sum,
		Buckets:           bks,
		ExplicitBounds:    bounds,
	}
}
func getSummaryDataPoint(lbls []*commonpb.StringKeyValue, sum float64, count uint64, pcts []float64, values []float64) *otlp.SummaryDataPoint {
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
		TimeUnixNano:      uint64(time.Now().Unix()),
		Count:             count,
		Sum:               sum,
		PercentileValues:  pcs,
	}
}

