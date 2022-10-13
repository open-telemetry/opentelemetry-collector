// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by "model/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "go run model/internal/cmd/pdatagen/main.go".

package internal

import (
	"go.opentelemetry.io/collector/pdata/internal/data"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

type ResourceMetricsSlice struct {
	orig *[]*otlpmetrics.ResourceMetrics
}

func GetOrigResourceMetricsSlice(ms ResourceMetricsSlice) *[]*otlpmetrics.ResourceMetrics {
	return ms.orig
}

func NewResourceMetricsSlice(orig *[]*otlpmetrics.ResourceMetrics) ResourceMetricsSlice {
	return ResourceMetricsSlice{orig: orig}
}

type ResourceMetrics struct {
	orig *otlpmetrics.ResourceMetrics
}

func GetOrigResourceMetrics(ms ResourceMetrics) *otlpmetrics.ResourceMetrics {
	return ms.orig
}

func NewResourceMetrics(orig *otlpmetrics.ResourceMetrics) ResourceMetrics {
	return ResourceMetrics{orig: orig}
}

type ScopeMetricsSlice struct {
	orig *[]*otlpmetrics.ScopeMetrics
}

func GetOrigScopeMetricsSlice(ms ScopeMetricsSlice) *[]*otlpmetrics.ScopeMetrics {
	return ms.orig
}

func NewScopeMetricsSlice(orig *[]*otlpmetrics.ScopeMetrics) ScopeMetricsSlice {
	return ScopeMetricsSlice{orig: orig}
}

type ScopeMetrics struct {
	orig *otlpmetrics.ScopeMetrics
}

func GetOrigScopeMetrics(ms ScopeMetrics) *otlpmetrics.ScopeMetrics {
	return ms.orig
}

func NewScopeMetrics(orig *otlpmetrics.ScopeMetrics) ScopeMetrics {
	return ScopeMetrics{orig: orig}
}

type MetricSlice struct {
	orig *[]*otlpmetrics.Metric
}

func GetOrigMetricSlice(ms MetricSlice) *[]*otlpmetrics.Metric {
	return ms.orig
}

func NewMetricSlice(orig *[]*otlpmetrics.Metric) MetricSlice {
	return MetricSlice{orig: orig}
}

type Metric struct {
	orig *otlpmetrics.Metric
}

func GetOrigMetric(ms Metric) *otlpmetrics.Metric {
	return ms.orig
}

func NewMetric(orig *otlpmetrics.Metric) Metric {
	return Metric{orig: orig}
}

type Gauge struct {
	orig *otlpmetrics.Gauge
}

func GetOrigGauge(ms Gauge) *otlpmetrics.Gauge {
	return ms.orig
}

func NewGauge(orig *otlpmetrics.Gauge) Gauge {
	return Gauge{orig: orig}
}

type Sum struct {
	orig *otlpmetrics.Sum
}

func GetOrigSum(ms Sum) *otlpmetrics.Sum {
	return ms.orig
}

func NewSum(orig *otlpmetrics.Sum) Sum {
	return Sum{orig: orig}
}

type Histogram struct {
	orig *otlpmetrics.Histogram
}

func GetOrigHistogram(ms Histogram) *otlpmetrics.Histogram {
	return ms.orig
}

func NewHistogram(orig *otlpmetrics.Histogram) Histogram {
	return Histogram{orig: orig}
}

type ExponentialHistogram struct {
	orig *otlpmetrics.ExponentialHistogram
}

func GetOrigExponentialHistogram(ms ExponentialHistogram) *otlpmetrics.ExponentialHistogram {
	return ms.orig
}

func NewExponentialHistogram(orig *otlpmetrics.ExponentialHistogram) ExponentialHistogram {
	return ExponentialHistogram{orig: orig}
}

type Summary struct {
	orig *otlpmetrics.Summary
}

func GetOrigSummary(ms Summary) *otlpmetrics.Summary {
	return ms.orig
}

func NewSummary(orig *otlpmetrics.Summary) Summary {
	return Summary{orig: orig}
}

type NumberDataPointSlice struct {
	orig *[]*otlpmetrics.NumberDataPoint
}

func GetOrigNumberDataPointSlice(ms NumberDataPointSlice) *[]*otlpmetrics.NumberDataPoint {
	return ms.orig
}

func NewNumberDataPointSlice(orig *[]*otlpmetrics.NumberDataPoint) NumberDataPointSlice {
	return NumberDataPointSlice{orig: orig}
}

type NumberDataPoint struct {
	orig *otlpmetrics.NumberDataPoint
}

func GetOrigNumberDataPoint(ms NumberDataPoint) *otlpmetrics.NumberDataPoint {
	return ms.orig
}

func NewNumberDataPoint(orig *otlpmetrics.NumberDataPoint) NumberDataPoint {
	return NumberDataPoint{orig: orig}
}

type HistogramDataPointSlice struct {
	orig *[]*otlpmetrics.HistogramDataPoint
}

func GetOrigHistogramDataPointSlice(ms HistogramDataPointSlice) *[]*otlpmetrics.HistogramDataPoint {
	return ms.orig
}

func NewHistogramDataPointSlice(orig *[]*otlpmetrics.HistogramDataPoint) HistogramDataPointSlice {
	return HistogramDataPointSlice{orig: orig}
}

type HistogramDataPoint struct {
	orig *otlpmetrics.HistogramDataPoint
}

func GetOrigHistogramDataPoint(ms HistogramDataPoint) *otlpmetrics.HistogramDataPoint {
	return ms.orig
}

func NewHistogramDataPoint(orig *otlpmetrics.HistogramDataPoint) HistogramDataPoint {
	return HistogramDataPoint{orig: orig}
}

type ExponentialHistogramDataPointSlice struct {
	orig *[]*otlpmetrics.ExponentialHistogramDataPoint
}

func GetOrigExponentialHistogramDataPointSlice(ms ExponentialHistogramDataPointSlice) *[]*otlpmetrics.ExponentialHistogramDataPoint {
	return ms.orig
}

func NewExponentialHistogramDataPointSlice(orig *[]*otlpmetrics.ExponentialHistogramDataPoint) ExponentialHistogramDataPointSlice {
	return ExponentialHistogramDataPointSlice{orig: orig}
}

type ExponentialHistogramDataPoint struct {
	orig *otlpmetrics.ExponentialHistogramDataPoint
}

func GetOrigExponentialHistogramDataPoint(ms ExponentialHistogramDataPoint) *otlpmetrics.ExponentialHistogramDataPoint {
	return ms.orig
}

func NewExponentialHistogramDataPoint(orig *otlpmetrics.ExponentialHistogramDataPoint) ExponentialHistogramDataPoint {
	return ExponentialHistogramDataPoint{orig: orig}
}

type ExponentialHistogramDataPointBuckets struct {
	orig *otlpmetrics.ExponentialHistogramDataPoint_Buckets
}

func GetOrigExponentialHistogramDataPointBuckets(ms ExponentialHistogramDataPointBuckets) *otlpmetrics.ExponentialHistogramDataPoint_Buckets {
	return ms.orig
}

func NewExponentialHistogramDataPointBuckets(orig *otlpmetrics.ExponentialHistogramDataPoint_Buckets) ExponentialHistogramDataPointBuckets {
	return ExponentialHistogramDataPointBuckets{orig: orig}
}

type SummaryDataPointSlice struct {
	orig *[]*otlpmetrics.SummaryDataPoint
}

func GetOrigSummaryDataPointSlice(ms SummaryDataPointSlice) *[]*otlpmetrics.SummaryDataPoint {
	return ms.orig
}

func NewSummaryDataPointSlice(orig *[]*otlpmetrics.SummaryDataPoint) SummaryDataPointSlice {
	return SummaryDataPointSlice{orig: orig}
}

type SummaryDataPoint struct {
	orig *otlpmetrics.SummaryDataPoint
}

func GetOrigSummaryDataPoint(ms SummaryDataPoint) *otlpmetrics.SummaryDataPoint {
	return ms.orig
}

func NewSummaryDataPoint(orig *otlpmetrics.SummaryDataPoint) SummaryDataPoint {
	return SummaryDataPoint{orig: orig}
}

type SummaryDataPointValueAtQuantileSlice struct {
	orig *[]*otlpmetrics.SummaryDataPoint_ValueAtQuantile
}

func GetOrigSummaryDataPointValueAtQuantileSlice(ms SummaryDataPointValueAtQuantileSlice) *[]*otlpmetrics.SummaryDataPoint_ValueAtQuantile {
	return ms.orig
}

func NewSummaryDataPointValueAtQuantileSlice(orig *[]*otlpmetrics.SummaryDataPoint_ValueAtQuantile) SummaryDataPointValueAtQuantileSlice {
	return SummaryDataPointValueAtQuantileSlice{orig: orig}
}

type SummaryDataPointValueAtQuantile struct {
	orig *otlpmetrics.SummaryDataPoint_ValueAtQuantile
}

func GetOrigSummaryDataPointValueAtQuantile(ms SummaryDataPointValueAtQuantile) *otlpmetrics.SummaryDataPoint_ValueAtQuantile {
	return ms.orig
}

func NewSummaryDataPointValueAtQuantile(orig *otlpmetrics.SummaryDataPoint_ValueAtQuantile) SummaryDataPointValueAtQuantile {
	return SummaryDataPointValueAtQuantile{orig: orig}
}

type ExemplarSlice struct {
	orig *[]otlpmetrics.Exemplar
}

func GetOrigExemplarSlice(ms ExemplarSlice) *[]otlpmetrics.Exemplar {
	return ms.orig
}

func NewExemplarSlice(orig *[]otlpmetrics.Exemplar) ExemplarSlice {
	return ExemplarSlice{orig: orig}
}

type Exemplar struct {
	orig *otlpmetrics.Exemplar
}

func GetOrigExemplar(ms Exemplar) *otlpmetrics.Exemplar {
	return ms.orig
}

func NewExemplar(orig *otlpmetrics.Exemplar) Exemplar {
	return Exemplar{orig: orig}
}

func GenerateTestResourceMetricsSlice() ResourceMetricsSlice {
	orig := []*otlpmetrics.ResourceMetrics{}
	tv := NewResourceMetricsSlice(&orig)
	FillTestResourceMetricsSlice(tv)
	return tv
}

func FillTestResourceMetricsSlice(tv ResourceMetricsSlice) {
	*tv.orig = make([]*otlpmetrics.ResourceMetrics, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.ResourceMetrics{}
		FillTestResourceMetrics(NewResourceMetrics((*tv.orig)[i]))
	}
}

func GenerateTestResourceMetrics() ResourceMetrics {
	orig := otlpmetrics.ResourceMetrics{}
	tv := NewResourceMetrics(&orig)
	FillTestResourceMetrics(tv)
	return tv
}

func FillTestResourceMetrics(tv ResourceMetrics) {
	FillTestResource(NewResource(&tv.orig.Resource))
	tv.orig.SchemaUrl = "https://opentelemetry.io/schemas/1.5.0"
	FillTestScopeMetricsSlice(NewScopeMetricsSlice(&tv.orig.ScopeMetrics))
}

func GenerateTestScopeMetricsSlice() ScopeMetricsSlice {
	orig := []*otlpmetrics.ScopeMetrics{}
	tv := NewScopeMetricsSlice(&orig)
	FillTestScopeMetricsSlice(tv)
	return tv
}

func FillTestScopeMetricsSlice(tv ScopeMetricsSlice) {
	*tv.orig = make([]*otlpmetrics.ScopeMetrics, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.ScopeMetrics{}
		FillTestScopeMetrics(NewScopeMetrics((*tv.orig)[i]))
	}
}

func GenerateTestScopeMetrics() ScopeMetrics {
	orig := otlpmetrics.ScopeMetrics{}
	tv := NewScopeMetrics(&orig)
	FillTestScopeMetrics(tv)
	return tv
}

func FillTestScopeMetrics(tv ScopeMetrics) {
	FillTestInstrumentationScope(NewInstrumentationScope(&tv.orig.Scope))
	tv.orig.SchemaUrl = "https://opentelemetry.io/schemas/1.5.0"
	FillTestMetricSlice(NewMetricSlice(&tv.orig.Metrics))
}

func GenerateTestMetricSlice() MetricSlice {
	orig := []*otlpmetrics.Metric{}
	tv := NewMetricSlice(&orig)
	FillTestMetricSlice(tv)
	return tv
}

func FillTestMetricSlice(tv MetricSlice) {
	*tv.orig = make([]*otlpmetrics.Metric, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.Metric{}
		FillTestMetric(NewMetric((*tv.orig)[i]))
	}
}

func GenerateTestMetric() Metric {
	orig := otlpmetrics.Metric{}
	tv := NewMetric(&orig)
	FillTestMetric(tv)
	return tv
}

func FillTestMetric(tv Metric) {
	tv.orig.Name = "test_name"
	tv.orig.Description = "test_description"
	tv.orig.Unit = "1"
	tv.orig.Data = &otlpmetrics.Metric_Sum{Sum: &otlpmetrics.Sum{}}
	FillTestSum(NewSum(tv.orig.GetSum()))
}

func GenerateTestGauge() Gauge {
	orig := otlpmetrics.Gauge{}
	tv := NewGauge(&orig)
	FillTestGauge(tv)
	return tv
}

func FillTestGauge(tv Gauge) {
	FillTestNumberDataPointSlice(NewNumberDataPointSlice(&tv.orig.DataPoints))
}

func GenerateTestSum() Sum {
	orig := otlpmetrics.Sum{}
	tv := NewSum(&orig)
	FillTestSum(tv)
	return tv
}

func FillTestSum(tv Sum) {
	tv.orig.AggregationTemporality = otlpmetrics.AggregationTemporality(1)
	tv.orig.IsMonotonic = true
	FillTestNumberDataPointSlice(NewNumberDataPointSlice(&tv.orig.DataPoints))
}

func GenerateTestHistogram() Histogram {
	orig := otlpmetrics.Histogram{}
	tv := NewHistogram(&orig)
	FillTestHistogram(tv)
	return tv
}

func FillTestHistogram(tv Histogram) {
	tv.orig.AggregationTemporality = otlpmetrics.AggregationTemporality(1)
	FillTestHistogramDataPointSlice(NewHistogramDataPointSlice(&tv.orig.DataPoints))
}

func GenerateTestExponentialHistogram() ExponentialHistogram {
	orig := otlpmetrics.ExponentialHistogram{}
	tv := NewExponentialHistogram(&orig)
	FillTestExponentialHistogram(tv)
	return tv
}

func FillTestExponentialHistogram(tv ExponentialHistogram) {
	tv.orig.AggregationTemporality = otlpmetrics.AggregationTemporality(1)
	FillTestExponentialHistogramDataPointSlice(NewExponentialHistogramDataPointSlice(&tv.orig.DataPoints))
}

func GenerateTestSummary() Summary {
	orig := otlpmetrics.Summary{}
	tv := NewSummary(&orig)
	FillTestSummary(tv)
	return tv
}

func FillTestSummary(tv Summary) {
	FillTestSummaryDataPointSlice(NewSummaryDataPointSlice(&tv.orig.DataPoints))
}

func GenerateTestNumberDataPointSlice() NumberDataPointSlice {
	orig := []*otlpmetrics.NumberDataPoint{}
	tv := NewNumberDataPointSlice(&orig)
	FillTestNumberDataPointSlice(tv)
	return tv
}

func FillTestNumberDataPointSlice(tv NumberDataPointSlice) {
	*tv.orig = make([]*otlpmetrics.NumberDataPoint, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.NumberDataPoint{}
		FillTestNumberDataPoint(NewNumberDataPoint((*tv.orig)[i]))
	}
}

func GenerateTestNumberDataPoint() NumberDataPoint {
	orig := otlpmetrics.NumberDataPoint{}
	tv := NewNumberDataPoint(&orig)
	FillTestNumberDataPoint(tv)
	return tv
}

func FillTestNumberDataPoint(tv NumberDataPoint) {
	FillTestMap(NewMap(&tv.orig.Attributes))
	tv.orig.StartTimeUnixNano = 1234567890
	tv.orig.TimeUnixNano = 1234567890
	tv.orig.Value = &otlpmetrics.NumberDataPoint_AsDouble{AsDouble: float64(17.13)}
	FillTestExemplarSlice(NewExemplarSlice(&tv.orig.Exemplars))
	tv.orig.Flags = 1
}

func GenerateTestHistogramDataPointSlice() HistogramDataPointSlice {
	orig := []*otlpmetrics.HistogramDataPoint{}
	tv := NewHistogramDataPointSlice(&orig)
	FillTestHistogramDataPointSlice(tv)
	return tv
}

func FillTestHistogramDataPointSlice(tv HistogramDataPointSlice) {
	*tv.orig = make([]*otlpmetrics.HistogramDataPoint, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.HistogramDataPoint{}
		FillTestHistogramDataPoint(NewHistogramDataPoint((*tv.orig)[i]))
	}
}

func GenerateTestHistogramDataPoint() HistogramDataPoint {
	orig := otlpmetrics.HistogramDataPoint{}
	tv := NewHistogramDataPoint(&orig)
	FillTestHistogramDataPoint(tv)
	return tv
}

func FillTestHistogramDataPoint(tv HistogramDataPoint) {
	FillTestMap(NewMap(&tv.orig.Attributes))
	tv.orig.StartTimeUnixNano = 1234567890
	tv.orig.TimeUnixNano = 1234567890
	tv.orig.Count = uint64(17)
	tv.orig.Sum_ = &otlpmetrics.HistogramDataPoint_Sum{Sum: float64(17.13)}
	tv.orig.BucketCounts = []uint64{1, 2, 3}
	tv.orig.ExplicitBounds = []float64{1, 2, 3}
	FillTestExemplarSlice(NewExemplarSlice(&tv.orig.Exemplars))
	tv.orig.Flags = 1
	tv.orig.Min_ = &otlpmetrics.HistogramDataPoint_Min{Min: float64(9.23)}
	tv.orig.Max_ = &otlpmetrics.HistogramDataPoint_Max{Max: float64(182.55)}
}

func GenerateTestExponentialHistogramDataPointSlice() ExponentialHistogramDataPointSlice {
	orig := []*otlpmetrics.ExponentialHistogramDataPoint{}
	tv := NewExponentialHistogramDataPointSlice(&orig)
	FillTestExponentialHistogramDataPointSlice(tv)
	return tv
}

func FillTestExponentialHistogramDataPointSlice(tv ExponentialHistogramDataPointSlice) {
	*tv.orig = make([]*otlpmetrics.ExponentialHistogramDataPoint, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.ExponentialHistogramDataPoint{}
		FillTestExponentialHistogramDataPoint(NewExponentialHistogramDataPoint((*tv.orig)[i]))
	}
}

func GenerateTestExponentialHistogramDataPoint() ExponentialHistogramDataPoint {
	orig := otlpmetrics.ExponentialHistogramDataPoint{}
	tv := NewExponentialHistogramDataPoint(&orig)
	FillTestExponentialHistogramDataPoint(tv)
	return tv
}

func FillTestExponentialHistogramDataPoint(tv ExponentialHistogramDataPoint) {
	FillTestMap(NewMap(&tv.orig.Attributes))
	tv.orig.StartTimeUnixNano = 1234567890
	tv.orig.TimeUnixNano = 1234567890
	tv.orig.Count = uint64(17)
	tv.orig.Sum_ = &otlpmetrics.ExponentialHistogramDataPoint_Sum{Sum: float64(17.13)}
	tv.orig.Scale = int32(4)
	tv.orig.ZeroCount = uint64(201)
	FillTestExponentialHistogramDataPointBuckets(NewExponentialHistogramDataPointBuckets(&tv.orig.Positive))
	FillTestExponentialHistogramDataPointBuckets(NewExponentialHistogramDataPointBuckets(&tv.orig.Negative))
	FillTestExemplarSlice(NewExemplarSlice(&tv.orig.Exemplars))
	tv.orig.Flags = 1
	tv.orig.Min_ = &otlpmetrics.ExponentialHistogramDataPoint_Min{Min: float64(9.23)}
	tv.orig.Max_ = &otlpmetrics.ExponentialHistogramDataPoint_Max{Max: float64(182.55)}
}

func GenerateTestExponentialHistogramDataPointBuckets() ExponentialHistogramDataPointBuckets {
	orig := otlpmetrics.ExponentialHistogramDataPoint_Buckets{}
	tv := NewExponentialHistogramDataPointBuckets(&orig)
	FillTestExponentialHistogramDataPointBuckets(tv)
	return tv
}

func FillTestExponentialHistogramDataPointBuckets(tv ExponentialHistogramDataPointBuckets) {
	tv.orig.Offset = int32(909)
	tv.orig.BucketCounts = []uint64{1, 2, 3}
}

func GenerateTestSummaryDataPointSlice() SummaryDataPointSlice {
	orig := []*otlpmetrics.SummaryDataPoint{}
	tv := NewSummaryDataPointSlice(&orig)
	FillTestSummaryDataPointSlice(tv)
	return tv
}

func FillTestSummaryDataPointSlice(tv SummaryDataPointSlice) {
	*tv.orig = make([]*otlpmetrics.SummaryDataPoint, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.SummaryDataPoint{}
		FillTestSummaryDataPoint(NewSummaryDataPoint((*tv.orig)[i]))
	}
}

func GenerateTestSummaryDataPoint() SummaryDataPoint {
	orig := otlpmetrics.SummaryDataPoint{}
	tv := NewSummaryDataPoint(&orig)
	FillTestSummaryDataPoint(tv)
	return tv
}

func FillTestSummaryDataPoint(tv SummaryDataPoint) {
	FillTestMap(NewMap(&tv.orig.Attributes))
	tv.orig.StartTimeUnixNano = 1234567890
	tv.orig.TimeUnixNano = 1234567890
	tv.orig.Count = uint64(17)
	tv.orig.Sum = float64(17.13)
	FillTestSummaryDataPointValueAtQuantileSlice(NewSummaryDataPointValueAtQuantileSlice(&tv.orig.QuantileValues))
	tv.orig.Flags = 1
}

func GenerateTestSummaryDataPointValueAtQuantileSlice() SummaryDataPointValueAtQuantileSlice {
	orig := []*otlpmetrics.SummaryDataPoint_ValueAtQuantile{}
	tv := NewSummaryDataPointValueAtQuantileSlice(&orig)
	FillTestSummaryDataPointValueAtQuantileSlice(tv)
	return tv
}

func FillTestSummaryDataPointValueAtQuantileSlice(tv SummaryDataPointValueAtQuantileSlice) {
	*tv.orig = make([]*otlpmetrics.SummaryDataPoint_ValueAtQuantile, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.SummaryDataPoint_ValueAtQuantile{}
		FillTestSummaryDataPointValueAtQuantile(NewSummaryDataPointValueAtQuantile((*tv.orig)[i]))
	}
}

func GenerateTestSummaryDataPointValueAtQuantile() SummaryDataPointValueAtQuantile {
	orig := otlpmetrics.SummaryDataPoint_ValueAtQuantile{}
	tv := NewSummaryDataPointValueAtQuantile(&orig)
	FillTestSummaryDataPointValueAtQuantile(tv)
	return tv
}

func FillTestSummaryDataPointValueAtQuantile(tv SummaryDataPointValueAtQuantile) {
	tv.orig.Quantile = float64(17.13)
	tv.orig.Value = float64(17.13)
}

func GenerateTestExemplarSlice() ExemplarSlice {
	orig := []otlpmetrics.Exemplar{}
	tv := NewExemplarSlice(&orig)
	FillTestExemplarSlice(tv)
	return tv
}

func FillTestExemplarSlice(tv ExemplarSlice) {
	*tv.orig = make([]otlpmetrics.Exemplar, 7)
	for i := 0; i < 7; i++ {
		FillTestExemplar(NewExemplar(&(*tv.orig)[i]))
	}
}

func GenerateTestExemplar() Exemplar {
	orig := otlpmetrics.Exemplar{}
	tv := NewExemplar(&orig)
	FillTestExemplar(tv)
	return tv
}

func FillTestExemplar(tv Exemplar) {
	tv.orig.TimeUnixNano = 1234567890
	tv.orig.Value = &otlpmetrics.Exemplar_AsInt{AsInt: int64(17)}
	FillTestMap(NewMap(&tv.orig.FilteredAttributes))
	tv.orig.TraceId = data.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	tv.orig.SpanId = data.SpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1})
}
