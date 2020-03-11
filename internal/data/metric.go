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
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetric "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
)

// This file defines in-memory data structures to represent metrics.
// For the proto representation see https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/metrics/v1/metrics.proto

// MetricData is the top-level struct that is propagated through the metrics pipeline.
// This is the newer version of consumerdata.MetricsData, but uses more efficient
// in-memory representation.
//
// This is a reference type (like builtin map).
//
// Must use NewMetricData functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type MetricData struct {
	orig []*otlpmetric.ResourceMetrics

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalMetricData
}

type internalMetricData struct {
	resourceMetrics []ResourceMetrics
}

// NewMetricData creates a new MetricData.
func NewMetricData() MetricData {
	return MetricData{nil, &internalMetricData{}}
}

func (md MetricData) ResourceMetrics() []ResourceMetrics {
	return md.pimpl.resourceMetrics
}

func (md MetricData) SetResourceMetrics(r []ResourceMetrics) {
	md.pimpl.resourceMetrics = r
}

// MetricCount calculates the total number of metrics.
func (md MetricData) MetricCount() int {
	metricCount := 0
	// TODO: Do not access internal members, add a metricCount to ResourceMetrics.
	for _, rm := range md.pimpl.resourceMetrics {
		metricCount += len(rm.pimpl.metrics)
	}
	return metricCount
}

// ResourceMetrics is a collection of metrics from a Resource.
//
// Must use NewResourceMetrics functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceMetrics struct {
	orig *otlpmetric.ResourceMetrics

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalResourceMetrics
}

type internalResourceMetrics struct {
	resource *Resource
	metrics  []Metric
}

// NewResourceMetrics creates a new ResourceMetrics.
func NewResourceMetrics() ResourceMetrics {
	return ResourceMetrics{&otlpmetric.ResourceMetrics{}, &internalResourceMetrics{}}
}

func (rm ResourceMetrics) Resource() *Resource {
	return rm.pimpl.resource
}

func (rm ResourceMetrics) SetResource(r *Resource) {
	rm.pimpl.resource = r
}

func (rm ResourceMetrics) Metrics() []Metric {
	return rm.pimpl.metrics
}

func (rm ResourceMetrics) SetMetrics(s []Metric) {
	rm.pimpl.metrics = s
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
	pimpl *internalMetric
}

type internalMetric struct {
	metricDescriptor    MetricDescriptor
	int64DataPoints     []Int64DataPoint
	doubleDataPoints    []DoubleDataPoint
	histogramDataPoints []HistogramDataPoint
	summaryDataPoints   []SummaryDataPoint
}

// NewMetric creates a new Metric.
func NewMetric() Metric {
	return Metric{&otlpmetric.Metric{}, &internalMetric{}}
}

// NewMetricSlice creates a slice of Metrics that are correctly initialized.
func NewMetricSlice(len int) []Metric {
	// Slice for underlying orig.
	origs := make([]otlpmetric.Metric, len)
	// Slice for underlying pimpl.
	internals := make([]internalMetric, len)
	// Slice for wrappers.
	wrappers := make([]Metric, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &internals[i]
	}
	return wrappers
}

func (m Metric) MetricDescriptor() MetricDescriptor {
	return m.pimpl.metricDescriptor
}

func (m Metric) SetMetricDescriptor(r MetricDescriptor) {
	m.pimpl.metricDescriptor = r
}

func (m Metric) Int64DataPoints() []Int64DataPoint {
	return m.pimpl.int64DataPoints
}

func (m Metric) SetInt64DataPoints(v []Int64DataPoint) {
	m.pimpl.int64DataPoints = v
}

func (m Metric) DoubleDataPoints() []DoubleDataPoint {
	return m.pimpl.doubleDataPoints
}

func (m Metric) SetDoubleDataPoints(v []DoubleDataPoint) {
	m.pimpl.doubleDataPoints = v
}

func (m Metric) HistogramDataPoints() []HistogramDataPoint {
	return m.pimpl.histogramDataPoints
}

func (m Metric) SetHistogramDataPoints(v []HistogramDataPoint) {
	m.pimpl.histogramDataPoints = v
}

func (m Metric) SummaryDataPoints() []SummaryDataPoint {
	return m.pimpl.summaryDataPoints
}

func (m Metric) SetSummaryDataPoints(v []SummaryDataPoint) {
	m.pimpl.summaryDataPoints = v
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
	pimpl *internalMetricDescriptor
}

type internalMetricDescriptor struct {
	labels Labels
}

// NewMetricDescriptor creates a new MetricDescriptor.
func NewMetricDescriptor() MetricDescriptor {
	return MetricDescriptor{&otlpmetric.MetricDescriptor{}, &internalMetricDescriptor{}}
}

func (md MetricDescriptor) Name() string {
	return md.orig.Name
}

func (md MetricDescriptor) SetName(v string) {
	md.orig.Name = v
}

func (md MetricDescriptor) Description() string {
	return md.orig.Description
}

func (md MetricDescriptor) SetDescription(v string) {
	md.orig.Description = v
}

func (md MetricDescriptor) Unit() string {
	return md.orig.Unit
}

func (md MetricDescriptor) SetUnit(v string) {
	md.orig.Unit = v
}

func (md MetricDescriptor) Type() MetricType {
	return MetricType(md.orig.Type)
}

func (md MetricDescriptor) SetMetricType(v MetricType) {
	md.orig.Type = otlpmetric.MetricDescriptor_Type(v)
}

func (md MetricDescriptor) Labels() Labels {
	return md.pimpl.labels
}

func (md MetricDescriptor) SetLabels(v Labels) {
	md.pimpl.labels = v
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
	pimpl *internalInt64DataPoint
}

type internalInt64DataPoint struct {
	labels Labels
}

// NewInt64DataPoint creates a new Int64DataPoint
func NewInt64DataPoint() Int64DataPoint {
	return Int64DataPoint{&otlpmetric.Int64DataPoint{}, &internalInt64DataPoint{}}
}

// NewInt64DataPointSlice creates a slice of Int64DataPoint that are correctly initialized.
func NewInt64DataPointSlice(len int) []Int64DataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetric.Int64DataPoint, len)
	// Slice for underlying pimpl.
	internals := make([]internalInt64DataPoint, len)
	// Slice for wrappers.
	wrappers := make([]Int64DataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &internals[i]
	}
	return wrappers
}

func (dp Int64DataPoint) Labels() Labels {
	return dp.pimpl.labels
}

func (dp Int64DataPoint) SetLabels(v Labels) {
	dp.pimpl.labels = v
}

func (dp Int64DataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp Int64DataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp Int64DataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp Int64DataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp Int64DataPoint) Value() int64 {
	return dp.orig.Value
}

func (dp Int64DataPoint) SetValue(v int64) {
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
	pimpl *internalDoubleDataPoint
}

type internalDoubleDataPoint struct {
	labels Labels
}

// NewDoubleDataPoint creates a new DoubleDataPoint.
func NewDoubleDataPoint() *DoubleDataPoint {
	return &DoubleDataPoint{&otlpmetric.DoubleDataPoint{}, &internalDoubleDataPoint{}}
}

// NewDoubleDataPointSlice creates a slice of DoubleDataPoint that are correctly initialized.
func NewDoubleDataPointSlice(len int) []DoubleDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetric.DoubleDataPoint, len)
	// Slice for underlying pimpl.
	internals := make([]internalDoubleDataPoint, len)
	// Slice for wrappers.
	wrappers := make([]DoubleDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &internals[i]
	}
	return wrappers
}

func (dp DoubleDataPoint) Labels() Labels {
	return dp.pimpl.labels
}

func (dp DoubleDataPoint) SetLabels(v Labels) {
	dp.pimpl.labels = v
}

func (dp DoubleDataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp DoubleDataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp DoubleDataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp DoubleDataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp DoubleDataPoint) Value() float64 {
	return dp.orig.Value
}

func (dp DoubleDataPoint) SetValue(v float64) {
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
	pimpl *internalHistogramDataPoint
}

type internalHistogramDataPoint struct {
	labels  Labels
	buckets []HistogramBucket
}

// NewHistogramDataPoint creates a new HistogramDataPoint.
func NewHistogramDataPoint() HistogramDataPoint {
	return HistogramDataPoint{&otlpmetric.HistogramDataPoint{}, &internalHistogramDataPoint{}}
}

// NewHistogramDataPointSlice creates a slice of HistogramDataPoint that are correctly initialized.
func NewHistogramDataPointSlice(len int) []HistogramDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetric.HistogramDataPoint, len)
	// Slice for underlying pimpl.
	internals := make([]internalHistogramDataPoint, len)
	// Slice for wrappers.
	wrappers := make([]HistogramDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &internals[i]
	}
	return wrappers
}

func (dp HistogramDataPoint) Labels() Labels {
	return dp.pimpl.labels
}

func (dp HistogramDataPoint) SetLabels(v Labels) {
	dp.pimpl.labels = v
}

func (dp HistogramDataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp HistogramDataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp HistogramDataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp HistogramDataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp HistogramDataPoint) Count() uint64 {
	return dp.orig.Count
}

func (dp HistogramDataPoint) SetCount(v uint64) {
	dp.orig.Count = v
}

func (dp HistogramDataPoint) Sum() float64 {
	return dp.orig.Sum
}

func (dp HistogramDataPoint) SetSum(v float64) {
	dp.orig.Sum = v
}

func (dp HistogramDataPoint) Buckets() []HistogramBucket {
	return dp.pimpl.buckets
}

func (dp HistogramDataPoint) SetBuckets(v []HistogramBucket) {
	dp.pimpl.buckets = v
}

func (dp HistogramDataPoint) ExplicitBounds() []float64 {
	return dp.orig.ExplicitBounds
}

func (dp HistogramDataPoint) SetExplicitBounds(v []float64) {
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
	pimpl *internalHistogramBucket
}

type internalHistogramBucket struct {
	exemplar HistogramBucketExemplar
}

// NewHistogramBucket creates a new HistogramBucket.
func NewHistogramBucket() HistogramBucket {
	return HistogramBucket{&otlpmetric.HistogramDataPoint_Bucket{}, &internalHistogramBucket{}}
}

// NewHistogramBucketSlice creates a slice of HistogramBucket that are correctly initialized.
func NewHistogramBucketSlice(len int) []HistogramBucket {
	// Slice for underlying orig.
	origs := make([]otlpmetric.HistogramDataPoint_Bucket, len)
	// Slice for underlying pimpl.
	internals := make([]internalHistogramBucket, len)
	// Slice for wrappers.
	wrappers := make([]HistogramBucket, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &internals[i]
	}
	return wrappers
}

func (hb HistogramBucket) Count() uint64 {
	return hb.orig.Count
}

func (hb HistogramBucket) SetCount(v uint64) {
	hb.orig.Count = v
}

func (hb HistogramBucket) Exemplar() HistogramBucketExemplar {
	return hb.pimpl.exemplar
}

func (hb HistogramBucket) SetExemplar(v HistogramBucketExemplar) {
	hb.pimpl.exemplar = v
}

// HistogramBucketExemplar are example points that may be used to annotate aggregated Histogram values.
// They are metadata that gives information about a particular value added to a Histogram bucket.
//
// Must use NewHistogramBucketExemplar* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramBucketExemplar struct {
	// Wrap OTLP HistogramDataPoint_Bucket_Exemplar.
	orig *otlpmetric.HistogramDataPoint_Bucket_Exemplar

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *intenalHistogramBucketExemplar
}

type intenalHistogramBucketExemplar struct {
	attachments Labels
}

// NewHistogramBucketExemplar creates a new HistogramBucketExemplar.
func NewHistogramBucketExemplar() HistogramBucketExemplar {
	return HistogramBucketExemplar{&otlpmetric.HistogramDataPoint_Bucket_Exemplar{}, &intenalHistogramBucketExemplar{}}
}

func (hbe HistogramBucketExemplar) Value() float64 {
	return hbe.orig.Value
}

func (hbe HistogramBucketExemplar) SetValue(v float64) {
	hbe.orig.Value = v
}

func (hbe HistogramBucketExemplar) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(hbe.orig.TimestampUnixnano)
}

func (hbe HistogramBucketExemplar) SetTimestamp(v TimestampUnixNano) {
	hbe.orig.TimestampUnixnano = uint64(v)
}

func (hbe HistogramBucketExemplar) Attachments() Labels {
	return hbe.pimpl.attachments
}

func (hbe HistogramBucketExemplar) SetAttachments(v Labels) {
	hbe.pimpl.attachments = v
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
	pimpl *intenalSummaryDataPoint
}

type intenalSummaryDataPoint struct {
	labels             Labels
	valueAtPercentiles []SummaryValueAtPercentile
}

// NewSummaryDataPoint creates a new SummaryDataPoint.
func NewSummaryDataPoint() SummaryDataPoint {
	return SummaryDataPoint{&otlpmetric.SummaryDataPoint{}, &intenalSummaryDataPoint{}}
}

// NewSummaryDataPointSlice creates a slice of SummaryDataPoint that are correctly initialized.
func NewSummaryDataPointSlice(len int) []SummaryDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetric.SummaryDataPoint, len)
	// Slice for underlying pimpl.
	internals := make([]intenalSummaryDataPoint, len)
	// Slice for wrappers.
	wrappers := make([]SummaryDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &internals[i]
	}
	return wrappers
}

func (dp SummaryDataPoint) Labels() Labels {
	return dp.pimpl.labels
}

func (dp SummaryDataPoint) SetLabels(v Labels) {
	dp.pimpl.labels = v
}

func (dp SummaryDataPoint) StartTime() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.StartTimeUnixnano)
}

func (dp SummaryDataPoint) SetStartTime(v TimestampUnixNano) {
	dp.orig.StartTimeUnixnano = uint64(v)
}

func (dp SummaryDataPoint) Timestamp() TimestampUnixNano {
	return TimestampUnixNano(dp.orig.TimestampUnixnano)
}

func (dp SummaryDataPoint) SetTimestamp(v TimestampUnixNano) {
	dp.orig.TimestampUnixnano = uint64(v)
}

func (dp SummaryDataPoint) Count() uint64 {
	return dp.orig.Count
}

func (dp SummaryDataPoint) SetCount(v uint64) {
	dp.orig.Count = v
}

func (dp SummaryDataPoint) Sum() float64 {
	return dp.orig.Sum
}

func (dp SummaryDataPoint) SetSum(v float64) {
	dp.orig.Sum = v
}

func (dp SummaryDataPoint) ValueAtPercentiles() []SummaryValueAtPercentile {
	return dp.pimpl.valueAtPercentiles
}

func (dp SummaryDataPoint) SetValueAtPercentiles(v []SummaryValueAtPercentile) {
	dp.pimpl.valueAtPercentiles = v
}

// SummaryValueAtPercentile represents the value at a given percentile of a distribution.
//
// Must use NewSummaryValueAtPercentile* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type SummaryValueAtPercentile struct {
	// Wrap OTLP SummaryDataPoint_ValueAtPercentile.
	orig *otlpmetric.SummaryDataPoint_ValueAtPercentile
}

// NewSummaryValueAtPercentile creates a new SummaryValueAtPercentile.
func NewSummaryValueAtPercentile() SummaryValueAtPercentile {
	return SummaryValueAtPercentile{&otlpmetric.SummaryDataPoint_ValueAtPercentile{}}
}

// NewSummaryValueAtPercentileSlice creates a slice of SummaryValueAtPercentile that are correctly initialized.
func NewSummaryValueAtPercentileSlice(len int) []SummaryValueAtPercentile {
	// Slice for underlying orig.
	origs := make([]otlpmetric.SummaryDataPoint_ValueAtPercentile, len)
	// Slice for wrappers.
	wrappers := make([]SummaryValueAtPercentile, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func (vp SummaryValueAtPercentile) Percentile() float64 {
	return vp.orig.Percentile
}

func (vp SummaryValueAtPercentile) SetPercentile(v float64) {
	vp.orig.Percentile = v
}

func (vp SummaryValueAtPercentile) Value() float64 {
	return vp.orig.Value
}

func (vp SummaryValueAtPercentile) SetValue(v float64) {
	vp.orig.Value = v
}

// Labels stores the original representation of the labels and the internal LabelsMap representation.
type Labels struct {
	orig []*otlpcommon.StringKeyValue

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalLabel
}

type internalLabel struct {
	labelsMap LabelsMap
}

// NewLabels creates a new Labels.
func NewLabels() Labels {
	return Labels{nil, &internalLabel{}}
}

func (ls Labels) LabelsMap() LabelsMap {
	return ls.pimpl.labelsMap
}

func (ls Labels) SetLabelsMap(v LabelsMap) {
	ls.pimpl.labelsMap = v
}

// LabelsMap stores a map of label keys to values.
type LabelsMap map[string]string
