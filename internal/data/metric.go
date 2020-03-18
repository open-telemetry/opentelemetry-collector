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
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
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
	orig []*otlpmetrics.ResourceMetrics

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalMetricData
}

type internalMetricData struct {
	resourceMetrics []ResourceMetrics
	// True if the resourceMetrics was initialized.
	initializedSlice bool
}

// MetricDataFromOtlp creates the internal MetricData representation from the OTLP.
func MetricDataFromOtlp(orig []*otlpmetrics.ResourceMetrics) MetricData {
	return MetricData{orig, &internalMetricData{}}
}

// MetricDataToOtlp converts the internal MetricData to the OTLP.
func MetricDataToOtlp(md MetricData) []*otlpmetrics.ResourceMetrics {
	return md.toOrig()
}

// NewMetricData creates a new MetricData.
func NewMetricData() MetricData {
	return MetricData{nil, &internalMetricData{}}
}

func (md MetricData) ResourceMetrics() []ResourceMetrics {
	if !md.pimpl.initializedSlice {
		md.pimpl.resourceMetrics = newResourceMetricsSliceFromOrig(md.orig)
		md.pimpl.initializedSlice = true
	}
	return md.pimpl.resourceMetrics
}

func (md MetricData) SetResourceMetrics(r []ResourceMetrics) {
	md.pimpl.resourceMetrics = r
	md.pimpl.initializedSlice = true

	if len(md.pimpl.resourceMetrics) == 0 {
		md.orig = nil
		return
	}

	// Reconstruct the slice because we don't know what elements were removed/added.
	md.orig = make([]*otlpmetrics.ResourceMetrics, len(md.pimpl.resourceMetrics))
	for i := range md.pimpl.resourceMetrics {
		md.orig[i] = md.pimpl.resourceMetrics[i].orig
	}
}

// MetricCount calculates the total number of metrics.
func (md MetricData) MetricCount() int {
	metricCount := 0
	for _, rm := range md.ResourceMetrics() {
		metricCount += rm.MetricCount()
	}
	return metricCount
}

func (md MetricData) toOrig() []*otlpmetrics.ResourceMetrics {
	if !md.pimpl.initializedSlice {
		// Guaranteed no changes in the resourceMetrics fields.
		return md.orig
	}

	// User may have changed internal fields in any ResourceMetrics, flush all of them.
	for i := range md.pimpl.resourceMetrics {
		md.pimpl.resourceMetrics[i].flushInternal()
	}

	return md.orig
}

// ResourceMetrics is a collection of metrics from a Resource.
//
// Must use NewResourceMetrics functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceMetrics struct {
	orig *otlpmetrics.ResourceMetrics

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalResourceMetrics
}

type internalResourceMetrics struct {
	resource                      *Resource
	instrumentationLibraryMetrics []InstrumentationLibraryMetrics
	// True if the pimpl was initialized.
	initialized bool
}

// NewResourceMetrics creates a new ResourceMetrics.
func NewResourceMetrics() ResourceMetrics {
	return ResourceMetrics{&otlpmetrics.ResourceMetrics{}, &internalResourceMetrics{}}
}

// NewResourceMetricsSlice creates a slice of ResourceMetrics that are correctly initialized.
func NewResourceMetricsSlice(len int) []ResourceMetrics {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.ResourceMetrics, len)
	// Slice for underlying pimpl.
	pimpls := make([]internalResourceMetrics, len)
	// Slice for wrappers.
	wrappers := make([]ResourceMetrics, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newResourceMetricsSliceFromOrig(origs []*otlpmetrics.ResourceMetrics) []ResourceMetrics {
	// Slice for underlying pimpl.
	pimpls := make([]internalResourceMetrics, len(origs))
	// Slice for wrappers.
	wrappers := make([]ResourceMetrics, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (rm ResourceMetrics) Resource() *Resource {
	rm.initInternallIfNeeded()
	return rm.pimpl.resource
}

func (rm ResourceMetrics) SetResource(r *Resource) {
	rm.initInternallIfNeeded()
	rm.pimpl.resource = r
}

func (rm ResourceMetrics) InstrumentationLibraryMetrics() []InstrumentationLibraryMetrics {
	rm.initInternallIfNeeded()
	return rm.pimpl.instrumentationLibraryMetrics
}

func (rm ResourceMetrics) SetInstrumentationLibraryMetrics(s []InstrumentationLibraryMetrics) {
	rm.initInternallIfNeeded()
	rm.pimpl.instrumentationLibraryMetrics = s

	if len(rm.pimpl.instrumentationLibraryMetrics) == 0 {
		rm.orig.InstrumentationLibraryMetrics = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	rm.orig.InstrumentationLibraryMetrics = make([]*otlpmetrics.InstrumentationLibraryMetrics, len(rm.pimpl.instrumentationLibraryMetrics))
	for i := range rm.pimpl.instrumentationLibraryMetrics {
		rm.orig.InstrumentationLibraryMetrics[i] = rm.pimpl.instrumentationLibraryMetrics[i].orig
	}
}

// MetricCount calculates the total number of metrics.
func (rm ResourceMetrics) MetricCount() int {
	metricCount := 0
	for _, ilm := range rm.InstrumentationLibraryMetrics() {
		metricCount += len(ilm.pimpl.metrics)
	}
	return metricCount
}

func (rm ResourceMetrics) flushInternal() {
	if !rm.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}

	rm.orig.Resource = rm.pimpl.resource.toOrig()
}

func (rm ResourceMetrics) initInternallIfNeeded() {
	if !rm.pimpl.initialized {
		rm.pimpl.resource = newResource(rm.orig.Resource)
		rm.pimpl.instrumentationLibraryMetrics = newInstrumentationLibraryMetricsSliceFromOrig(rm.orig.InstrumentationLibraryMetrics)
		rm.pimpl.initialized = true
	}
}

// InstrumentationLibraryMetrics is a collection of metrics from a Resource.
//
// Must use NewResourceMetrics functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibraryMetrics struct {
	// Wrap OTLP InstrumentationLibraryMetric.
	orig *otlpmetrics.InstrumentationLibraryMetrics

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalInstrumentationLibraryMetrics
}

type internalInstrumentationLibraryMetrics struct {
	metrics []Metric
	// True if the metrics was initialized.
	initializedSlice bool
}

// NewInstrumentationLibraryMetricsSlice creates a slice of InstrumentationLibraryMetrics that are correctly initialized.
func NewInstrumentationLibraryMetricsSlice(len int) []InstrumentationLibraryMetrics {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.InstrumentationLibraryMetrics, len)
	// Slice for underlying pimpl.
	pimpls := make([]internalInstrumentationLibraryMetrics, len)
	// Slice for wrappers.
	wrappers := make([]InstrumentationLibraryMetrics, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newInstrumentationLibraryMetricsSliceFromOrig(origs []*otlpmetrics.InstrumentationLibraryMetrics) []InstrumentationLibraryMetrics {
	// Slice for underlying pimpl.
	pimpls := make([]internalInstrumentationLibraryMetrics, len(origs))
	// Slice for wrappers.
	wrappers := make([]InstrumentationLibraryMetrics, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (ilm InstrumentationLibraryMetrics) InstrumentationLibrary() InstrumentationLibrary {
	return newInstrumentationLibrary(ilm.orig.InstrumentationLibrary)
}

func (ilm InstrumentationLibraryMetrics) SetInstrumentationLibrary(il InstrumentationLibrary) {
	ilm.orig.InstrumentationLibrary = il.orig
}

func (ilm InstrumentationLibraryMetrics) Metrics() []Metric {
	if !ilm.pimpl.initializedSlice {
		ilm.pimpl.metrics = newMetricSliceFromOrig(ilm.orig.Metrics)
		ilm.pimpl.initializedSlice = true
	}
	return ilm.pimpl.metrics
}

func (ilm InstrumentationLibraryMetrics) SetMetrics(ms []Metric) {
	ilm.pimpl.metrics = ms
	ilm.pimpl.initializedSlice = true

	if len(ilm.pimpl.metrics) == 0 {
		ilm.orig.Metrics = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	ilm.orig.Metrics = make([]*otlpmetrics.Metric, len(ilm.pimpl.metrics))
	for i := range ilm.pimpl.metrics {
		ilm.orig.Metrics[i] = ilm.pimpl.metrics[i].orig
	}
}

// Metric defines a metric which has a descriptor and one or more timeseries points.
//
// Must use NewMetric* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Metric struct {
	// Wrap OTLP Metric.
	orig *otlpmetrics.Metric

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalMetric
}

type internalMetric struct {
	int64DataPoints     []Int64DataPoint
	doubleDataPoints    []DoubleDataPoint
	histogramDataPoints []HistogramDataPoint
	summaryDataPoints   []SummaryDataPoint
	// True if any slice was initialized.
	initializedSlices bool
}

// NewMetric creates a new Metric.
func NewMetric() Metric {
	return Metric{&otlpmetrics.Metric{}, &internalMetric{}}
}

// NewMetricSlice creates a slice of Metrics that are correctly initialized.
func NewMetricSlice(len int) []Metric {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.Metric, len)
	// Slice for underlying pimpl.
	pimpls := make([]internalMetric, len)
	// Slice for wrappers.
	wrappers := make([]Metric, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newMetricSliceFromOrig(origs []*otlpmetrics.Metric) []Metric {
	// Slice for underlying pimpl.
	pimpls := make([]internalMetric, len(origs))
	// Slice for wrappers.
	wrappers := make([]Metric, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (m Metric) MetricDescriptor() MetricDescriptor {
	if m.orig.MetricDescriptor == nil {
		// No MetricDescriptor available, initialize one to make all operations available.
		m.orig.MetricDescriptor = &otlpmetrics.MetricDescriptor{}
	}
	return newMetricDescriptor(m.orig.MetricDescriptor)
}

func (m Metric) SetMetricDescriptor(md MetricDescriptor) {
	m.orig.MetricDescriptor = md.orig
}

func (m Metric) Int64DataPoints() []Int64DataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.int64DataPoints
}

func (m Metric) SetInt64DataPoints(v []Int64DataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.int64DataPoints = v

	if len(m.pimpl.int64DataPoints) == 0 {
		m.orig.Int64DataPoints = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	m.orig.Int64DataPoints = make([]*otlpmetrics.Int64DataPoint, len(m.pimpl.int64DataPoints))
	for i := range m.pimpl.int64DataPoints {
		m.orig.Int64DataPoints[i] = m.pimpl.int64DataPoints[i].orig
	}
}

func (m Metric) DoubleDataPoints() []DoubleDataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.doubleDataPoints
}

func (m Metric) SetDoubleDataPoints(v []DoubleDataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.doubleDataPoints = v

	if len(m.pimpl.doubleDataPoints) == 0 {
		m.orig.DoubleDataPoints = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	m.orig.DoubleDataPoints = make([]*otlpmetrics.DoubleDataPoint, len(m.pimpl.doubleDataPoints))
	for i := range m.pimpl.doubleDataPoints {
		m.orig.DoubleDataPoints[i] = m.pimpl.doubleDataPoints[i].orig
	}
}

func (m Metric) HistogramDataPoints() []HistogramDataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.histogramDataPoints
}

func (m Metric) SetHistogramDataPoints(v []HistogramDataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.histogramDataPoints = v

	if len(m.pimpl.histogramDataPoints) == 0 {
		m.orig.HistogramDataPoints = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	m.orig.HistogramDataPoints = make([]*otlpmetrics.HistogramDataPoint, len(m.pimpl.histogramDataPoints))
	for i := range m.pimpl.histogramDataPoints {
		m.orig.HistogramDataPoints[i] = m.pimpl.histogramDataPoints[i].orig
	}

}

func (m Metric) SummaryDataPoints() []SummaryDataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.summaryDataPoints
}

func (m Metric) SetSummaryDataPoints(v []SummaryDataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.summaryDataPoints = v

	if len(m.pimpl.summaryDataPoints) == 0 {
		m.orig.SummaryDataPoints = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	m.orig.SummaryDataPoints = make([]*otlpmetrics.SummaryDataPoint, len(m.pimpl.summaryDataPoints))
	for i := range m.pimpl.summaryDataPoints {
		m.orig.SummaryDataPoints[i] = m.pimpl.summaryDataPoints[i].orig
	}
}

func (m Metric) initInternallIfNeeded() {
	if !m.pimpl.initializedSlices {
		if len(m.orig.Int64DataPoints) != 0 {
			m.pimpl.int64DataPoints = newInt64DataPointSliceFromOrig(m.orig.Int64DataPoints)
		}
		if len(m.orig.DoubleDataPoints) != 0 {
			m.pimpl.doubleDataPoints = newDoubleDataPointSliceFormOrgig(m.orig.DoubleDataPoints)
		}
		if len(m.orig.HistogramDataPoints) != 0 {
			m.pimpl.histogramDataPoints = newHistogramDataPointSliceFromOrig(m.orig.HistogramDataPoints)
		}
		if len(m.orig.SummaryDataPoints) != 0 {
			m.pimpl.summaryDataPoints = newSummaryDataPointSliceFromOrig(m.orig.SummaryDataPoints)
		}
		m.pimpl.initializedSlices = true
	}
}

// MetricDescriptor is the descriptor of a metric.
//
// Must use NewMetricDescriptor* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type MetricDescriptor struct {
	// Wrap OTLP MetricDescriptor.
	orig *otlpmetrics.MetricDescriptor
}

// NewMetricDescriptor creates a new MetricDescriptor.
func NewMetricDescriptor() MetricDescriptor {
	return MetricDescriptor{&otlpmetrics.MetricDescriptor{}}
}

func newMetricDescriptor(orig *otlpmetrics.MetricDescriptor) MetricDescriptor {
	return MetricDescriptor{orig}
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
	md.orig.Type = otlpmetrics.MetricDescriptor_Type(v)
}

func (md MetricDescriptor) LabelsMap() StringMap {
	return newStringMap(&md.orig.Labels)
}

func (md MetricDescriptor) SetLabelsMap(sm StringMap) {
	md.orig.Labels = *sm.orig
}

type MetricType otlpmetrics.MetricDescriptor_Type

const (
	MetricTypeUnspecified         MetricType = MetricType(otlpmetrics.MetricDescriptor_UNSPECIFIED)
	MetricTypeGaugeInt64          MetricType = MetricType(otlpmetrics.MetricDescriptor_GAUGE_INT64)
	MetricTypeGaugeDouble         MetricType = MetricType(otlpmetrics.MetricDescriptor_GAUGE_DOUBLE)
	MetricTypeGaugeHistogram      MetricType = MetricType(otlpmetrics.MetricDescriptor_GAUGE_HISTOGRAM)
	MetricTypeCounterInt64        MetricType = MetricType(otlpmetrics.MetricDescriptor_COUNTER_INT64)
	MetricTypeCounterDouble       MetricType = MetricType(otlpmetrics.MetricDescriptor_COUNTER_DOUBLE)
	MetricTypeCumulativeHistogram MetricType = MetricType(otlpmetrics.MetricDescriptor_CUMULATIVE_HISTOGRAM)
	MetricTypeSummary             MetricType = MetricType(otlpmetrics.MetricDescriptor_SUMMARY)
)

// Int64DataPoint is a single data point in a timeseries that describes the time-varying
// values of a int64 metric.
//
// Must use NewInt64DataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Int64DataPoint struct {
	// Wrap OTLP Int64DataPoint.
	orig *otlpmetrics.Int64DataPoint
}

// NewInt64DataPoint creates a new Int64DataPoint
func NewInt64DataPoint() Int64DataPoint {
	return Int64DataPoint{&otlpmetrics.Int64DataPoint{}}
}

// NewInt64DataPointSlice creates a slice of Int64DataPoint that are correctly initialized.
func NewInt64DataPointSlice(len int) []Int64DataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.Int64DataPoint, len)
	// Slice for wrappers.
	wrappers := make([]Int64DataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func newInt64DataPointSliceFromOrig(origs []*otlpmetrics.Int64DataPoint) []Int64DataPoint {
	// Slice for wrappers.
	wrappers := make([]Int64DataPoint, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
	}
	return wrappers
}

func (dp Int64DataPoint) LabelsMap() StringMap {
	return newStringMap(&dp.orig.Labels)
}

func (dp Int64DataPoint) SetLabelsMap(sm StringMap) {
	dp.orig.Labels = *sm.orig
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
	orig *otlpmetrics.DoubleDataPoint
}

// NewDoubleDataPoint creates a new DoubleDataPoint.
func NewDoubleDataPoint() *DoubleDataPoint {
	return &DoubleDataPoint{&otlpmetrics.DoubleDataPoint{}}
}

// NewDoubleDataPointSlice creates a slice of DoubleDataPoint that are correctly initialized.
func NewDoubleDataPointSlice(len int) []DoubleDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.DoubleDataPoint, len)
	// Slice for wrappers.
	wrappers := make([]DoubleDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func newDoubleDataPointSliceFormOrgig(origs []*otlpmetrics.DoubleDataPoint) []DoubleDataPoint {
	// Slice for wrappers.
	wrappers := make([]DoubleDataPoint, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
	}
	return wrappers
}

func (dp DoubleDataPoint) LabelsMap() StringMap {
	return newStringMap(&dp.orig.Labels)
}

func (dp DoubleDataPoint) SetLabelsMap(sm StringMap) {
	dp.orig.Labels = *sm.orig
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
	orig *otlpmetrics.HistogramDataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalHistogramDataPoint
}

type internalHistogramDataPoint struct {
	buckets []HistogramBucket
	// True if the buckets slice was initialized.
	initializedSlice bool
}

// NewHistogramDataPoint creates a new HistogramDataPoint.
func NewHistogramDataPoint() HistogramDataPoint {
	return HistogramDataPoint{&otlpmetrics.HistogramDataPoint{}, &internalHistogramDataPoint{}}
}

// NewHistogramDataPointSlice creates a slice of HistogramDataPoint that are correctly initialized.
func NewHistogramDataPointSlice(len int) []HistogramDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.HistogramDataPoint, len)
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

func newHistogramDataPointSliceFromOrig(origs []*otlpmetrics.HistogramDataPoint) []HistogramDataPoint {
	// Slice for underlying pimpl.
	pimpls := make([]internalHistogramDataPoint, len(origs))
	// Slice for wrappers.
	wrappers := make([]HistogramDataPoint, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (dp HistogramDataPoint) LabelsMap() StringMap {
	return newStringMap(&dp.orig.Labels)
}

func (dp HistogramDataPoint) SetLabelsMap(sm StringMap) {
	dp.orig.Labels = *sm.orig
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
	if !dp.pimpl.initializedSlice {
		dp.pimpl.buckets = newHistogramBucketSliceFromOrig(dp.orig.Buckets)
		dp.pimpl.initializedSlice = true
	}
	return dp.pimpl.buckets
}

func (dp HistogramDataPoint) SetBuckets(v []HistogramBucket) {
	dp.pimpl.buckets = v
	dp.pimpl.initializedSlice = true
	if len(dp.pimpl.buckets) == 0 {
		dp.orig.Buckets = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	dp.orig.Buckets = make([]*otlpmetrics.HistogramDataPoint_Bucket, len(dp.pimpl.buckets))
	for i := range dp.pimpl.buckets {
		dp.orig.Buckets[i] = dp.pimpl.buckets[i].orig
	}
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
	orig *otlpmetrics.HistogramDataPoint_Bucket
}

// NewHistogramBucket creates a new HistogramBucket.
func NewHistogramBucket() HistogramBucket {
	return HistogramBucket{&otlpmetrics.HistogramDataPoint_Bucket{}}
}

// NewHistogramBucketSlice creates a slice of HistogramBucket that are correctly initialized.
func NewHistogramBucketSlice(len int) []HistogramBucket {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.HistogramDataPoint_Bucket, len)
	// Slice for wrappers.
	wrappers := make([]HistogramBucket, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func newHistogramBucketSliceFromOrig(origs []*otlpmetrics.HistogramDataPoint_Bucket) []HistogramBucket {
	// Slice for wrappers.
	wrappers := make([]HistogramBucket, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
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
	if hb.orig.Exemplar == nil {
		// No Exemplar available, initialize one to make all operations available.
		hb.orig.Exemplar = &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{}
	}
	return newHistogramBucketExemplar(hb.orig.Exemplar)
}

func (hb HistogramBucket) SetExemplar(v HistogramBucketExemplar) {
	hb.orig.Exemplar = v.orig
}

// HistogramBucketExemplar are example points that may be used to annotate aggregated Histogram values.
// They are metadata that gives information about a particular value added to a Histogram bucket.
//
// Must use NewHistogramBucketExemplar* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramBucketExemplar struct {
	// Wrap OTLP HistogramDataPoint_Bucket_Exemplar.
	orig *otlpmetrics.HistogramDataPoint_Bucket_Exemplar
}

// NewHistogramBucketExemplar creates a new HistogramBucketExemplar.
func NewHistogramBucketExemplar() HistogramBucketExemplar {
	return HistogramBucketExemplar{&otlpmetrics.HistogramDataPoint_Bucket_Exemplar{}}
}

func newHistogramBucketExemplar(orig *otlpmetrics.HistogramDataPoint_Bucket_Exemplar) HistogramBucketExemplar {
	return HistogramBucketExemplar{orig}
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

func (hbe HistogramBucketExemplar) Attachments() StringMap {
	return newStringMap(&hbe.orig.Attachments)
}

func (hbe HistogramBucketExemplar) SetAttachments(sm StringMap) {
	hbe.orig.Attachments = *sm.orig
}

// SummaryDataPoint is a single data point in a timeseries that describes the time-varying
// values of a Summary metric.
//
// Must use NewSummaryDataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type SummaryDataPoint struct {
	// Wrap OTLP SummaryDataPoint.
	orig *otlpmetrics.SummaryDataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *intenalSummaryDataPoint
}

type intenalSummaryDataPoint struct {
	valueAtPercentiles []SummaryValueAtPercentile
	// True if the valueAtPercentiles was initialized.
	initializedSlice bool
}

// NewSummaryDataPoint creates a new SummaryDataPoint.
func NewSummaryDataPoint() SummaryDataPoint {
	return SummaryDataPoint{&otlpmetrics.SummaryDataPoint{}, &intenalSummaryDataPoint{}}
}

// NewSummaryDataPointSlice creates a slice of SummaryDataPoint that are correctly initialized.
func NewSummaryDataPointSlice(len int) []SummaryDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.SummaryDataPoint, len)
	// Slice for underlying pimpl.
	pimpls := make([]intenalSummaryDataPoint, len)
	// Slice for wrappers.
	wrappers := make([]SummaryDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newSummaryDataPointSliceFromOrig(origs []*otlpmetrics.SummaryDataPoint) []SummaryDataPoint {
	// Slice for underlying pimpl.
	pimpls := make([]intenalSummaryDataPoint, len(origs))
	// Slice for wrappers.
	wrappers := make([]SummaryDataPoint, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (dp SummaryDataPoint) LabelsMap() StringMap {
	return newStringMap(&dp.orig.Labels)
}

func (dp SummaryDataPoint) SetLabelsMap(sm StringMap) {
	dp.orig.Labels = *sm.orig
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
	if !dp.pimpl.initializedSlice {
		dp.pimpl.valueAtPercentiles = newSummaryValueAtPercentileSliceFromOrig(dp.orig.PercentileValues)
		dp.pimpl.initializedSlice = true
	}
	return dp.pimpl.valueAtPercentiles
}

func (dp SummaryDataPoint) SetValueAtPercentiles(v []SummaryValueAtPercentile) {
	dp.pimpl.valueAtPercentiles = v
	dp.pimpl.initializedSlice = true
	if len(dp.pimpl.valueAtPercentiles) == 0 {
		dp.orig.PercentileValues = nil
		return
	}

	// TODO: reuse orig slice if capacity is enough.
	// Reconstruct the slice because we don't know what elements were removed/added.
	dp.orig.PercentileValues = make([]*otlpmetrics.SummaryDataPoint_ValueAtPercentile, len(dp.pimpl.valueAtPercentiles))
	for i := range dp.pimpl.valueAtPercentiles {
		dp.orig.PercentileValues[i] = dp.pimpl.valueAtPercentiles[i].orig
	}
}

// SummaryValueAtPercentile represents the value at a given percentile of a distribution.
//
// Must use NewSummaryValueAtPercentile* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type SummaryValueAtPercentile struct {
	// Wrap OTLP SummaryDataPoint_ValueAtPercentile.
	orig *otlpmetrics.SummaryDataPoint_ValueAtPercentile
}

// NewSummaryValueAtPercentile creates a new SummaryValueAtPercentile.
func NewSummaryValueAtPercentile() SummaryValueAtPercentile {
	return SummaryValueAtPercentile{&otlpmetrics.SummaryDataPoint_ValueAtPercentile{}}
}

// NewSummaryValueAtPercentileSlice creates a slice of SummaryValueAtPercentile that are correctly initialized.
func NewSummaryValueAtPercentileSlice(len int) []SummaryValueAtPercentile {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.SummaryDataPoint_ValueAtPercentile, len)
	// Slice for wrappers.
	wrappers := make([]SummaryValueAtPercentile, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

func newSummaryValueAtPercentileSliceFromOrig(origs []*otlpmetrics.SummaryDataPoint_ValueAtPercentile) []SummaryValueAtPercentile {
	// Slice for wrappers.
	wrappers := make([]SummaryValueAtPercentile, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
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
