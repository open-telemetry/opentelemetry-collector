// Copyright 2020 OpenTelemetry Authors
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

package data

import (
	otlpmetric "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
)

// This file defines in-memory data structures to represent metrics.
// For the proto representation see https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/metrics/v1/metrics.proto

// MetricData is the top-level struct that is propagated through the metrics pipeline.
// This is the newer version of consumerdata.MetricsData, but uses more efficient
// in-memory representation.
type MetricData struct {
	resourceMetrics []*ResourceMetrics
}

func NewMetricData(resourceMetrics []*ResourceMetrics) MetricData {
	return MetricData{resourceMetrics}
}

// MetricCount calculates the total number of metrics.
func (md MetricData) MetricCount() int {
	metricCount := 0
	for _, rml := range md.resourceMetrics {
		metricCount += len(rml.metrics)
	}
	return metricCount
}

func (md MetricData) ResourceMetrics() []*ResourceMetrics {
	return md.resourceMetrics
}

func (md MetricData) SetResource(r []*ResourceMetrics) {
	md.resourceMetrics = r
}

// A collection of metrics from a Resource.
//
// Must use NewResourceMetrics functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceMetrics struct {
	// The resource for the metrics in this message.
	// If this field is not set then no resource info is known.
	resource *Resource

	// A list of Spans that originate from a resource.
	metrics []*Metric
}

func NewResourceMetrics(resource *Resource, metrics []*Metric) *ResourceMetrics {
	return &ResourceMetrics{resource, metrics}
}

func (rm *ResourceMetrics) Resource() *Resource {
	return rm.resource
}

func (rm *ResourceMetrics) SetResource(r *Resource) {
	rm.resource = r
}

func (rm *ResourceMetrics) Metrics() []*Metric {
	return rm.metrics
}

func (rm *ResourceMetrics) SetMetrics(s []*Metric) {
	rm.metrics = s
}

// Metric defines a metric which has a descriptor and one or more timeseries points.
//
// Must use NewMetric* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Metric struct {
	// Wrap OTLP Metric.
	orig *otlpmetric.Metric

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	metricDescriptor    MetricDescriptor
	int64DataPoints     []*Int64DataPoint
	doubleDataPoints    []*DoubleDataPoint
	histogramDataPoints []*HistogramDataPoint
	summaryDataPoints   []*SummaryDataPoint
}

func NewMetric() *Metric {
	return &Metric{orig: &otlpmetric.Metric{}}
}

// NewMetricSlice creates a slice of Metrics that are correctly initialized.
func NewMetricSlice(len int) []Metric {
	origs := make([]otlpmetric.Metric, len)
	wrappers := make([]Metric, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (m *Metric) MetricDescriptor() MetricDescriptor {
	return m.metricDescriptor
}

func (m *Metric) SetMetricDescriptor(r MetricDescriptor) {
	m.metricDescriptor = r
}

func (m *Metric) Int64DataPoints() []*Int64DataPoint {
	return m.int64DataPoints
}

func (m *Metric) SetInt64DataPoints(v []*Int64DataPoint) {
	m.int64DataPoints = v
}

func (m *Metric) DoubleDataPoints() []*DoubleDataPoint {
	return m.doubleDataPoints
}

func (m *Metric) SetDoubleDataPoints(v []*DoubleDataPoint) {
	m.doubleDataPoints = v
}

func (m *Metric) HistogramDataPoints() []*HistogramDataPoint {
	return m.histogramDataPoints
}

func (m *Metric) SetHistogramDataPoints(v []*HistogramDataPoint) {
	m.histogramDataPoints = v
}

func (m *Metric) SummaryDataPoints() []*SummaryDataPoint {
	return m.summaryDataPoints
}

func (m *Metric) SetSummaryDataPoints(v []*SummaryDataPoint) {
	m.summaryDataPoints = v
}

// MetricDescriptor is the descriptor of a metric.
//
// Must use NewMetricDescriptor* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type MetricDescriptor struct {
	// Wrap OTLP MetricDescriptor.
	orig *otlpmetric.MetricDescriptor

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	labels Labels
}

func NewMetricDescriptor() *MetricDescriptor {
	return &MetricDescriptor{orig: &otlpmetric.MetricDescriptor{}}
}

func (md *MetricDescriptor) Name() string {
	return md.orig.Name
}

func (md *MetricDescriptor) SetName(v string) {
	md.orig.Name = v
}

func (md *MetricDescriptor) Description() string {
	return md.orig.Description
}

func (md *MetricDescriptor) SetDescription(v string) {
	md.orig.Description = v
}

func (md *MetricDescriptor) Unit() string {
	return md.orig.Unit
}

func (md *MetricDescriptor) SetUnit(v string) {
	md.orig.Unit = v
}

func (md *MetricDescriptor) Type() MetricType {
	return MetricType(md.orig.Type)
}

func (md *MetricDescriptor) SetMetricType(v MetricType) {
	md.orig.Type = otlpmetric.MetricDescriptor_Type(v)
}

func (md *MetricDescriptor) Labels() Labels {
	return md.labels
}

func (md *MetricDescriptor) SetLabels(v Labels) {
	md.labels = v
}

type MetricType otlpmetric.MetricDescriptor_Type

const (
	MetricTypeUnspecified         MetricType = MetricType(otlpmetric.MetricDescriptor_UNSPECIFIED)
	MetricTypeGaugeInt64          MetricType = MetricType(otlpmetric.MetricDescriptor_GAUGE_INT64)
	MetricTypeGaugeDouble         MetricType = MetricType(otlpmetric.MetricDescriptor_GAUGE_DOUBLE)
	MetricTypeGaugeHistogram      MetricType = MetricType(otlpmetric.MetricDescriptor_GAUGE_HISTOGRAM)
	MetricTypeCounterInt64        MetricType = MetricType(otlpmetric.MetricDescriptor_COUNTER_INT64)
	MetricTypeCounterDouble       MetricType = MetricType(otlpmetric.MetricDescriptor_COUNTER_DOUBLE)
	MetricTypeCumulativeHistogram MetricType = MetricType(otlpmetric.MetricDescriptor_CUMULATIVE_HISTOGRAM)
	MetricTypeSummary             MetricType = MetricType(otlpmetric.MetricDescriptor_SUMMARY)
)

// Int64DataPoint is a single data point in a timeseries that describes the time-varying
// values of a int64 metric.
//
// Must use NewInt64DataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Int64DataPoint struct {
	// Wrap OTLP Int64DataPoint.
	orig *otlpmetric.Int64DataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	labels Labels
}

func NewInt64DataPoint() *Int64DataPoint {
	return &Int64DataPoint{orig: &otlpmetric.Int64DataPoint{}}
}

// NewInt64DataPointSlice creates a slice of Int64DataPoint that are correctly initialized.
func NewInt64DataPointSlice(len int) []Int64DataPoint {
	origs := make([]otlpmetric.Int64DataPoint, len)
	wrappers := make([]Int64DataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (dp *Int64DataPoint) Labels() Labels {
	return dp.labels
}

func (dp *Int64DataPoint) SetLabels(v Labels) {
	dp.labels = v
}

func (dp *Int64DataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp *Int64DataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp *Int64DataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp *Int64DataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp *Int64DataPoint) Value() int64 {
	return dp.orig.Value
}

func (dp *Int64DataPoint) SetValue(v int64) {
	dp.orig.Value = v
}

// DoubleDataPoint is a single data point in a timeseries that describes the time-varying
// value of a double metric.
//
// Must use NewDoubleDataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type DoubleDataPoint struct {
	// Wrap OTLP DoubleDataPoint.
	orig *otlpmetric.DoubleDataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	labels Labels
}

func NewDoubleDataPoint() *DoubleDataPoint {
	return &DoubleDataPoint{orig: &otlpmetric.DoubleDataPoint{}}
}

// NewDoubleDataPointSlice creates a slice of DoubleDataPoint that are correctly initialized.
func NewDoubleDataPointSlice(len int) []DoubleDataPoint {
	origs := make([]otlpmetric.DoubleDataPoint, len)
	wrappers := make([]DoubleDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (dp *DoubleDataPoint) Labels() Labels {
	return dp.labels
}

func (dp *DoubleDataPoint) SetLabels(v Labels) {
	dp.labels = v
}

func (dp *DoubleDataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp *DoubleDataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp *DoubleDataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp *DoubleDataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp *DoubleDataPoint) Value() float64 {
	return dp.orig.Value
}

func (dp *DoubleDataPoint) SetValue(v float64) {
	dp.orig.Value = v
}

// HistogramDataPoint is a single data point in a timeseries that describes the time-varying
// values of a Histogram.
//
// Must use NewHistogramDataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramDataPoint struct {
	// Wrap OTLP HistogramDataPoint.
	orig *otlpmetric.HistogramDataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	labels  Labels
	buckets []*HistogramBucket
}

func NewHistogramDataPoint() *HistogramDataPoint {
	return &HistogramDataPoint{orig: &otlpmetric.HistogramDataPoint{}}
}

// NewHistogramDataPointSlice creates a slice of HistogramDataPoint that are correctly initialized.
func NewHistogramDataPointSlice(len int) []HistogramDataPoint {
	origs := make([]otlpmetric.HistogramDataPoint, len)
	wrappers := make([]HistogramDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (dp *HistogramDataPoint) Labels() Labels {
	return dp.labels
}

func (dp *HistogramDataPoint) SetLabels(v Labels) {
	dp.labels = v
}

func (dp *HistogramDataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp *HistogramDataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp *HistogramDataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp *HistogramDataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp *HistogramDataPoint) Count() uint64 {
	return dp.orig.Count
}

func (dp *HistogramDataPoint) SetCount(v uint64) {
	dp.orig.Count = v
}

func (dp *HistogramDataPoint) Sum() float64 {
	return dp.orig.Sum
}

func (dp *HistogramDataPoint) SetSum(v float64) {
	dp.orig.Sum = v
}

func (dp *HistogramDataPoint) Buckets() []*HistogramBucket {
	return dp.buckets
}

func (dp *HistogramDataPoint) SetBuckets(v []*HistogramBucket) {
	dp.buckets = v
}

func (dp *HistogramDataPoint) ExplicitBounds() []float64 {
	return dp.orig.ExplicitBounds
}

func (dp *HistogramDataPoint) SetExplicitBounds(v []float64) {
	dp.orig.ExplicitBounds = v
}

// HistogramBucket contains values for a histogram bucket.
//
// Must use NewHistogramBucket* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramBucket struct {
	// Wrap OTLP HistogramDataPoint_Bucket.
	orig *otlpmetric.HistogramDataPoint_Bucket

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	exemplar HistogramBucketExemplar
}

func NewHistogramBucket() *HistogramBucket {
	return &HistogramBucket{orig: &otlpmetric.HistogramDataPoint_Bucket{}}
}

// NewHistogramBucketSlice creates a slice of HistogramBucket that are correctly initialized.
func NewHistogramBucketSlice(len int) []HistogramBucket {
	origs := make([]otlpmetric.HistogramDataPoint_Bucket, len)
	wrappers := make([]HistogramBucket, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (hb *HistogramBucket) Count() uint64 {
	return hb.orig.Count
}

func (hb *HistogramBucket) SetCount(v uint64) {
	hb.orig.Count = v
}

func (hb *HistogramBucket) Exemplar() HistogramBucketExemplar {
	return hb.exemplar
}

func (hb *HistogramBucket) SetExemplar(v HistogramBucketExemplar) {
	hb.exemplar = v
}

// HistogramBucket are example points that may be used to annotate aggregated Histogram values.
// They are metadata that gives information about a particular value added to a Histogram bucket.
//
// Must use NewHistogramBucketExemplar* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramBucketExemplar struct {
	// Wrap OTLP HistogramDataPoint_Bucket_Exemplar.
	orig *otlpmetric.HistogramDataPoint_Bucket_Exemplar

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	attachments Labels
}

func NewHistogramBucketExemplar() *HistogramBucketExemplar {
	return &HistogramBucketExemplar{orig: &otlpmetric.HistogramDataPoint_Bucket_Exemplar{}}
}

func (hbe *HistogramBucketExemplar) Value() float64 {
	return hbe.orig.Value
}

func (hbe *HistogramBucketExemplar) SetValue(v float64) {
	hbe.orig.Value = v
}

func (hbe *HistogramBucketExemplar) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(hbe.orig.TimestampUnixnano)
}

func (hbe *HistogramBucketExemplar) SetTimestamp(v TimestampUnixNano) {
	hbe.orig.TimestampUnixnano = uint64(v)
}

func (hbe *HistogramBucketExemplar) Attachments() Labels {
	return hbe.attachments
}

func (hbe *HistogramBucketExemplar) SetAttachments(v Labels) {
	hbe.attachments = v
}

// SummaryDataPoint is a single data point in a timeseries that describes the time-varying
// values of a Summary metric.
//
// Must use NewSummaryDataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type SummaryDataPoint struct {
	// Wrap OTLP SummaryDataPoint.
	orig *otlpmetric.SummaryDataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	labels             Labels
	valueAtPercentiles []*SummaryValueAtPercentile
}

func NewSummaryDataPoint() *SummaryDataPoint {
	return &SummaryDataPoint{orig: &otlpmetric.SummaryDataPoint{}}
}

// NewSummaryDataPointSlice creates a slice of SummaryDataPoint that are correctly initialized.
func NewSummaryDataPointSlice(len int) []SummaryDataPoint {
	origs := make([]otlpmetric.SummaryDataPoint, len)
	wrappers := make([]SummaryDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (dp *SummaryDataPoint) Labels() Labels {
	return dp.labels
}

func (dp *SummaryDataPoint) SetLabels(v Labels) {
	dp.labels = v
}

func (dp *SummaryDataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp *SummaryDataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp *SummaryDataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp *SummaryDataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp *SummaryDataPoint) Count() uint64 {
	return dp.orig.Count
}

func (dp *SummaryDataPoint) SetCount(v uint64) {
	dp.orig.Count = v
}

func (dp *SummaryDataPoint) Sum() float64 {
	return dp.orig.Sum
}

func (dp *SummaryDataPoint) SetSum(v float64) {
	dp.orig.Sum = v
}

func (dp *SummaryDataPoint) ValueAtPercentiles() []*SummaryValueAtPercentile {
	return dp.valueAtPercentiles
}

func (dp *SummaryDataPoint) SetValueAtPercentiles(v []*SummaryValueAtPercentile) {
	dp.valueAtPercentiles = v
}

// SummaryValueAtPercentile represents the value at a given percentile of a distribution.
//
// Must use NewSummaryValueAtPercentile* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type SummaryValueAtPercentile struct {
	// Wrap OTLP SummaryDataPoint_ValueAtPercentile.
	orig *otlpmetric.SummaryDataPoint_ValueAtPercentile
}

func NewSummaryValueAtPercentile() *SummaryValueAtPercentile {
	return &SummaryValueAtPercentile{orig: &otlpmetric.SummaryDataPoint_ValueAtPercentile{}}
}

// NewSummaryValueAtPercentileSlice creates a slice of SummaryValueAtPercentile that are correctly initialized.
func NewSummaryValueAtPercentileSlice(len int) []SummaryValueAtPercentile {
	origs := make([]otlpmetric.SummaryDataPoint_ValueAtPercentile, len)
	wrappers := make([]SummaryValueAtPercentile, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (vp *SummaryValueAtPercentile) Percentile() float64 {
	return vp.orig.Percentile
}

func (vp *SummaryValueAtPercentile) SetPercentile(v float64) {
	vp.orig.Percentile = v
}

func (vp *SummaryValueAtPercentile) Value() float64 {
	return vp.orig.Value
}

func (vp *SummaryValueAtPercentile) SetValue(v float64) {
	vp.orig.Value = v
}

// Labels stores a map of label keys to values.
type Labels map[string]string
