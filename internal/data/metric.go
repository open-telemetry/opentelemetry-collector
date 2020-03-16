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
	// True when the slice was replace.
	sliceChanged bool
	// True if the pimpl was initialized.
	initialized bool
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
	md.initInternallIfNeeded()
	return md.pimpl.resourceMetrics
}

func (md MetricData) SetResourceMetrics(r []ResourceMetrics) {
	md.initInternallIfNeeded()
	md.pimpl.resourceMetrics = r
	md.pimpl.sliceChanged = true
}

// MetricCount calculates the total number of metrics.
func (md MetricData) MetricCount() int {
	metricCount := 0
	// TODO: Do not access internal members, add a metricCount to ResourceMetrics.
	for _, rm := range md.pimpl.resourceMetrics {
		for _, ilm := range rm.pimpl.instrumentationLibraryMetrics {
			metricCount += len(ilm.pimpl.metrics)
		}
	}
	return metricCount
}

func (md MetricData) toOrig() []*otlpmetrics.ResourceMetrics {
	if !md.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return md.orig
	}
	if md.pimpl.sliceChanged {
		// Reconstruct the slice because we don't know what elements were removed/added.
		// User may have changed internal fields in any ResourceMetrics, flush all of them.
		md.orig = make([]*otlpmetrics.ResourceMetrics, len(md.pimpl.resourceMetrics))
		for i := range md.pimpl.resourceMetrics {
			md.orig[i] = md.pimpl.resourceMetrics[i].getOrig()
			md.pimpl.resourceMetrics[i].flushInternal()
		}
	} else {
		// User may have changed internal fields in any ResourceMetrics, flush all of them.
		for i := range md.pimpl.resourceMetrics {
			md.pimpl.resourceMetrics[i].flushInternal()
		}
	}
	return md.orig
}

func (md MetricData) initInternallIfNeeded() {
	if !md.pimpl.initialized {
		md.pimpl.resourceMetrics = newResourceMetricsSliceFromOrig(md.orig)
		md.pimpl.initialized = true
	}
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
	// True when the slice was replace.
	sliceChanged bool
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
	// We don't update the orig slice because this may be called multiple times.
	rm.pimpl.sliceChanged = true
}

func (rm ResourceMetrics) getOrig() *otlpmetrics.ResourceMetrics {
	return rm.orig
}

func (rm ResourceMetrics) flushInternal() {
	if !rm.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}

	rm.orig.Resource = rm.pimpl.resource.toOrig()

	if rm.pimpl.sliceChanged {
		// Reconstruct the slice because we don't know what elements were removed/added.
		// User may have changed internal fields in any Metric, flush all of them.
		rm.orig.InstrumentationLibraryMetrics = make([]*otlpmetrics.InstrumentationLibraryMetrics, len(rm.pimpl.instrumentationLibraryMetrics))
		for i := range rm.pimpl.instrumentationLibraryMetrics {
			rm.orig.InstrumentationLibraryMetrics[i] = rm.pimpl.instrumentationLibraryMetrics[i].getOrig()
			rm.pimpl.instrumentationLibraryMetrics[i].flushInternal()
		}
	} else {
		// User may have changed internal fields in any Metric, flush all of them.
		for i := range rm.pimpl.instrumentationLibraryMetrics {
			rm.pimpl.instrumentationLibraryMetrics[i].flushInternal()
		}
	}
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
	instrumentationLibrary InstrumentationLibrary
	metrics                []Metric
	// True when the slice was replace.
	sliceChanged bool
	// True if the pimpl was initialized.
	initialized bool
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
	ilm.initInternallIfNeeded()
	return ilm.pimpl.instrumentationLibrary
}

func (ilm InstrumentationLibraryMetrics) SetResource(il InstrumentationLibrary) {
	ilm.initInternallIfNeeded()
	ilm.pimpl.instrumentationLibrary = il
}

func (ilm InstrumentationLibraryMetrics) Metrics() []Metric {
	ilm.initInternallIfNeeded()
	return ilm.pimpl.metrics
}

func (ilm InstrumentationLibraryMetrics) SetMetrics(ms []Metric) {
	ilm.initInternallIfNeeded()
	ilm.pimpl.metrics = ms
	// We don't update the orig slice because this may be called multiple times.
	ilm.pimpl.sliceChanged = true
}

func (ilm InstrumentationLibraryMetrics) initInternallIfNeeded() {
	if !ilm.pimpl.initialized {
		ilm.pimpl.instrumentationLibrary = newInstrumentationLibrary(ilm.orig.InstrumentationLibrary)
		ilm.pimpl.metrics = newMetricSliceFromOrig(ilm.orig.Metrics)
		ilm.pimpl.initialized = true
	}
}

func (ilm InstrumentationLibraryMetrics) getOrig() *otlpmetrics.InstrumentationLibraryMetrics {
	return ilm.orig
}

func (ilm InstrumentationLibraryMetrics) flushInternal() {
	if !ilm.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}

	if ilm.pimpl.sliceChanged {
		// Reconstruct the slice because we don't know what elements were removed/added.
		// User may have changed internal fields in any Metric, flush all of them.
		ilm.orig.Metrics = make([]*otlpmetrics.Metric, len(ilm.pimpl.metrics))
		for i := range ilm.pimpl.metrics {
			ilm.orig.Metrics[i] = ilm.pimpl.metrics[i].getOrig()
			ilm.pimpl.metrics[i].flushInternal()
		}
	} else {
		// User may have changed internal fields in any Metric, flush all of them.
		for i := range ilm.pimpl.metrics {
			ilm.pimpl.metrics[i].flushInternal()
		}
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
	metricDescriptor    MetricDescriptor
	int64DataPoints     []Int64DataPoint
	doubleDataPoints    []DoubleDataPoint
	histogramDataPoints []HistogramDataPoint
	summaryDataPoints   []SummaryDataPoint
	// True when the slice was replace.
	sliceChanged bool
	// True if the pimpl was initialized.
	initialized bool
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
	m.initInternallIfNeeded()
	return m.pimpl.metricDescriptor
}

func (m Metric) SetMetricDescriptor(r MetricDescriptor) {
	m.initInternallIfNeeded()
	m.pimpl.metricDescriptor = r
	m.orig.MetricDescriptor = r.getOrig()
}

func (m Metric) Int64DataPoints() []Int64DataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.int64DataPoints
}

func (m Metric) SetInt64DataPoints(v []Int64DataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.int64DataPoints = v
	m.pimpl.sliceChanged = true
}

func (m Metric) DoubleDataPoints() []DoubleDataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.doubleDataPoints
}

func (m Metric) SetDoubleDataPoints(v []DoubleDataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.doubleDataPoints = v
	m.pimpl.sliceChanged = true
}

func (m Metric) HistogramDataPoints() []HistogramDataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.histogramDataPoints
}

func (m Metric) SetHistogramDataPoints(v []HistogramDataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.histogramDataPoints = v
	m.pimpl.sliceChanged = true
}

func (m Metric) SummaryDataPoints() []SummaryDataPoint {
	m.initInternallIfNeeded()
	return m.pimpl.summaryDataPoints
}

func (m Metric) SetSummaryDataPoints(v []SummaryDataPoint) {
	m.initInternallIfNeeded()
	m.pimpl.summaryDataPoints = v
	m.pimpl.sliceChanged = true
}

func (m Metric) initInternallIfNeeded() {
	if !m.pimpl.initialized {
		m.pimpl.metricDescriptor = newMetricDescriptorFromOrig(m.orig.MetricDescriptor)
		m.pimpl.int64DataPoints = newInt64DataPointSliceFromOrig(m.orig.Int64DataPoints)
		m.pimpl.doubleDataPoints = newDoubleDataPointSliceFormOrgig(m.orig.DoubleDataPoints)
		m.pimpl.histogramDataPoints = newHistogramDataPointSliceFromOrig(m.orig.HistogramDataPoints)
		m.pimpl.summaryDataPoints = newSummaryDataPointSliceFromOrig(m.orig.SummaryDataPoints)
		m.pimpl.initialized = true
	}
}

func (m Metric) getOrig() *otlpmetrics.Metric {
	return m.orig
}

func (m Metric) flushInternal() {
	if !m.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}

	m.pimpl.metricDescriptor.flushInternal()

	if !m.pimpl.sliceChanged {
		// No slice structure changed. Just flush all the internal objects.
		m.pimpl.metricDescriptor.flushInternal()
		if len(m.pimpl.int64DataPoints) != 0 {
			for i := range m.pimpl.int64DataPoints {
				m.pimpl.int64DataPoints[i].flushInternal()
			}
		}
		if len(m.pimpl.doubleDataPoints) != 0 {
			for i := range m.pimpl.doubleDataPoints {
				m.pimpl.doubleDataPoints[i].flushInternal()
			}
		}
		if len(m.pimpl.histogramDataPoints) != 0 {
			for i := range m.pimpl.histogramDataPoints {
				m.pimpl.histogramDataPoints[i].flushInternal()
			}
		}
		if len(m.pimpl.summaryDataPoints) != 0 {
			for i := range m.pimpl.summaryDataPoints {
				m.pimpl.summaryDataPoints[i].flushInternal()
			}
		}
		return
	}

	if len(m.pimpl.int64DataPoints) != 0 {
		m.orig.Int64DataPoints = make([]*otlpmetrics.Int64DataPoint, len(m.pimpl.int64DataPoints))
		for i := range m.pimpl.int64DataPoints {
			m.orig.Int64DataPoints[i] = m.pimpl.int64DataPoints[i].getOrig()
			m.pimpl.int64DataPoints[i].flushInternal()
		}
	}

	if len(m.pimpl.doubleDataPoints) != 0 {
		m.orig.DoubleDataPoints = make([]*otlpmetrics.DoubleDataPoint, len(m.pimpl.doubleDataPoints))
		for i := range m.pimpl.doubleDataPoints {
			m.orig.DoubleDataPoints[i] = m.pimpl.doubleDataPoints[i].getOrig()
			m.pimpl.doubleDataPoints[i].flushInternal()
		}
	}

	if len(m.pimpl.histogramDataPoints) != 0 {
		m.orig.HistogramDataPoints = make([]*otlpmetrics.HistogramDataPoint, len(m.pimpl.histogramDataPoints))
		for i := range m.pimpl.histogramDataPoints {
			m.orig.HistogramDataPoints[i] = m.pimpl.histogramDataPoints[i].getOrig()
			m.pimpl.histogramDataPoints[i].flushInternal()
		}
	}

	if len(m.pimpl.summaryDataPoints) != 0 {
		m.orig.SummaryDataPoints = make([]*otlpmetrics.SummaryDataPoint, len(m.pimpl.summaryDataPoints))
		for i := range m.pimpl.summaryDataPoints {
			m.orig.SummaryDataPoints[i] = m.pimpl.summaryDataPoints[i].getOrig()
			m.pimpl.summaryDataPoints[i].flushInternal()
		}

	}

}

// MetricDescriptor is the descriptor of a metric.
//
// Must use NewMetricDescriptor* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type MetricDescriptor struct {
	// Wrap OTLP MetricDescriptor.
	orig *otlpmetrics.MetricDescriptor

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalMetricDescriptor
}

type internalMetricDescriptor struct {
	labels Labels
	// True if the pimpl was initialized.
	initialized bool
}

// NewMetricDescriptor creates a new MetricDescriptor.
func NewMetricDescriptor() MetricDescriptor {
	return MetricDescriptor{&otlpmetrics.MetricDescriptor{}, &internalMetricDescriptor{}}
}

func newMetricDescriptorFromOrig(orig *otlpmetrics.MetricDescriptor) MetricDescriptor {
	return MetricDescriptor{orig, &internalMetricDescriptor{}}
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

func (md MetricDescriptor) Labels() Labels {
	md.initInternallIfNeeded()
	return md.pimpl.labels
}

func (md MetricDescriptor) SetLabels(v Labels) {
	md.initInternallIfNeeded()
	md.pimpl.labels = v
}

func (md MetricDescriptor) initInternallIfNeeded() {
	if !md.pimpl.initialized {
		md.pimpl.labels = newLabelsFromOrig(md.orig.Labels)
		md.pimpl.initialized = true
	}
}

func (md MetricDescriptor) getOrig() *otlpmetrics.MetricDescriptor {
	return md.orig
}

func (md MetricDescriptor) flushInternal() {
	if !md.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	md.orig.Labels = md.pimpl.labels.toOrig()
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

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalInt64DataPoint
}

type internalInt64DataPoint struct {
	labels Labels
	// True if the pimpl was initialized.
	initialized bool
}

// NewInt64DataPoint creates a new Int64DataPoint
func NewInt64DataPoint() Int64DataPoint {
	return Int64DataPoint{&otlpmetrics.Int64DataPoint{}, &internalInt64DataPoint{}}
}

// NewInt64DataPointSlice creates a slice of Int64DataPoint that are correctly initialized.
func NewInt64DataPointSlice(len int) []Int64DataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.Int64DataPoint, len)
	// Slice for underlying pimpl.
	pimpls := make([]internalInt64DataPoint, len)
	// Slice for wrappers.
	wrappers := make([]Int64DataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newInt64DataPointSliceFromOrig(origs []*otlpmetrics.Int64DataPoint) []Int64DataPoint {
	// Slice for underlying pimpl.
	pimpls := make([]internalInt64DataPoint, len(origs))
	// Slice for wrappers.
	wrappers := make([]Int64DataPoint, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (dp Int64DataPoint) Labels() Labels {
	dp.initInternallIfNeeded()
	return dp.pimpl.labels
}

func (dp Int64DataPoint) SetLabels(v Labels) {
	dp.initInternallIfNeeded()
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

func (dp Int64DataPoint) initInternallIfNeeded() {
	if !dp.pimpl.initialized {
		dp.pimpl.labels = newLabelsFromOrig(dp.orig.Labels)
		dp.pimpl.initialized = true
	}
}

func (dp Int64DataPoint) getOrig() *otlpmetrics.Int64DataPoint {
	return dp.orig
}

func (dp Int64DataPoint) flushInternal() {
	if !dp.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	dp.orig.Labels = dp.pimpl.labels.toOrig()
}

// DoubleDataPoint is a single data point in a timeseries that describes the time-varying
// value of a double metric.
//
// Must use NewDoubleDataPoint* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type DoubleDataPoint struct {
	// Wrap OTLP DoubleDataPoint.
	orig *otlpmetrics.DoubleDataPoint

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalDoubleDataPoint
}

type internalDoubleDataPoint struct {
	labels Labels
	// True if the pimpl was initialized.
	initialized bool
}

// NewDoubleDataPoint creates a new DoubleDataPoint.
func NewDoubleDataPoint() *DoubleDataPoint {
	return &DoubleDataPoint{&otlpmetrics.DoubleDataPoint{}, &internalDoubleDataPoint{}}
}

// NewDoubleDataPointSlice creates a slice of DoubleDataPoint that are correctly initialized.
func NewDoubleDataPointSlice(len int) []DoubleDataPoint {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.DoubleDataPoint, len)
	// Slice for underlying pimpl.
	pimpls := make([]internalDoubleDataPoint, len)
	// Slice for wrappers.
	wrappers := make([]DoubleDataPoint, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newDoubleDataPointSliceFormOrgig(origs []*otlpmetrics.DoubleDataPoint) []DoubleDataPoint {
	// Slice for underlying pimpl.
	pimpls := make([]internalDoubleDataPoint, len(origs))
	// Slice for wrappers.
	wrappers := make([]DoubleDataPoint, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func (dp DoubleDataPoint) Labels() Labels {
	dp.initInternallIfNeeded()
	return dp.pimpl.labels
}

func (dp DoubleDataPoint) SetLabels(v Labels) {
	dp.initInternallIfNeeded()
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

func (dp DoubleDataPoint) initInternallIfNeeded() {
	if !dp.pimpl.initialized {
		dp.pimpl.labels = newLabelsFromOrig(dp.orig.Labels)
		dp.pimpl.initialized = true
	}
}

func (dp DoubleDataPoint) getOrig() *otlpmetrics.DoubleDataPoint {
	return dp.orig
}

func (dp DoubleDataPoint) flushInternal() {
	if !dp.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	dp.orig.Labels = dp.pimpl.labels.toOrig()
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
	labels  Labels
	buckets []HistogramBucket
	// True when the slice was replace.
	sliceChanged bool
	// True if the pimpl was initialized.
	initialized bool
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

func (dp HistogramDataPoint) Labels() Labels {
	dp.initInternallIfNeeded()
	return dp.pimpl.labels
}

func (dp HistogramDataPoint) SetLabels(v Labels) {
	dp.initInternallIfNeeded()
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
	dp.initInternallIfNeeded()
	return dp.pimpl.buckets
}

func (dp HistogramDataPoint) SetBuckets(v []HistogramBucket) {
	dp.initInternallIfNeeded()
	dp.pimpl.buckets = v
	dp.pimpl.sliceChanged = true
}

func (dp HistogramDataPoint) ExplicitBounds() []float64 {
	return dp.orig.ExplicitBounds
}

func (dp HistogramDataPoint) SetExplicitBounds(v []float64) {
	dp.orig.ExplicitBounds = v
}

func (dp HistogramDataPoint) initInternallIfNeeded() {
	if !dp.pimpl.initialized {
		dp.pimpl.labels = newLabelsFromOrig(dp.orig.Labels)
		dp.pimpl.buckets = newHistogramBucketSliceFromOrig(dp.orig.Buckets)
		dp.pimpl.initialized = true
	}
}

func (dp HistogramDataPoint) getOrig() *otlpmetrics.HistogramDataPoint {
	return dp.orig
}

func (dp HistogramDataPoint) flushInternal() {
	if !dp.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	dp.orig.Labels = dp.pimpl.labels.toOrig()

	if dp.pimpl.sliceChanged {
		// Reconstruct the slice because we don't know what elements were removed/added.
		// User may have changed internal fields in any ResourceMetrics, flush all of them.
		dp.orig.Buckets = make([]*otlpmetrics.HistogramDataPoint_Bucket, len(dp.pimpl.buckets))
		for i := range dp.pimpl.buckets {
			dp.orig.Buckets[i] = dp.pimpl.buckets[i].getOrig()
			dp.pimpl.buckets[i].flushInternal()
		}
	} else {
		// User may have changed internal fields in any ResourceMetrics, flush all of them.
		for i := range dp.pimpl.buckets {
			dp.pimpl.buckets[i].flushInternal()
		}
	}
}

// HistogramBucket contains values for a histogram bucket.
//
// Must use NewHistogramBucket* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramBucket struct {
	// Wrap OTLP HistogramDataPoint_Bucket.
	orig *otlpmetrics.HistogramDataPoint_Bucket

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalHistogramBucket
}

type internalHistogramBucket struct {
	exemplar HistogramBucketExemplar
	// True if the pimpl was initialized.
	initialized bool
}

// NewHistogramBucket creates a new HistogramBucket.
func NewHistogramBucket() HistogramBucket {
	return HistogramBucket{&otlpmetrics.HistogramDataPoint_Bucket{}, &internalHistogramBucket{}}
}

// NewHistogramBucketSlice creates a slice of HistogramBucket that are correctly initialized.
func NewHistogramBucketSlice(len int) []HistogramBucket {
	// Slice for underlying orig.
	origs := make([]otlpmetrics.HistogramDataPoint_Bucket, len)
	// Slice for underlying pimpl.
	pimpls := make([]internalHistogramBucket, len)
	// Slice for wrappers.
	wrappers := make([]HistogramBucket, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
		wrappers[i].pimpl = &pimpls[i]
	}
	return wrappers
}

func newHistogramBucketSliceFromOrig(origs []*otlpmetrics.HistogramDataPoint_Bucket) []HistogramBucket {
	// Slice for underlying pimpl.
	pimpls := make([]internalHistogramBucket, len(origs))
	// Slice for wrappers.
	wrappers := make([]HistogramBucket, len(origs))
	for i := range origs {
		wrappers[i].orig = origs[i]
		wrappers[i].pimpl = &pimpls[i]
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
	hb.initInternallIfNeeded()
	return hb.pimpl.exemplar
}

func (hb HistogramBucket) SetExemplar(v HistogramBucketExemplar) {
	hb.initInternallIfNeeded()
	hb.pimpl.exemplar = v
	hb.orig.Exemplar = v.getOrig()
}

func (hb HistogramBucket) initInternallIfNeeded() {
	if !hb.pimpl.initialized {
		hb.pimpl.exemplar = newHistogramBucketExemplarFromOrig(hb.orig.Exemplar)
		hb.pimpl.initialized = true
	}
}

func (hb HistogramBucket) getOrig() *otlpmetrics.HistogramDataPoint_Bucket {
	return hb.orig
}

func (hb HistogramBucket) flushInternal() {
	if !hb.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	hb.pimpl.exemplar.flushInternal()
}

// HistogramBucketExemplar are example points that may be used to annotate aggregated Histogram values.
// They are metadata that gives information about a particular value added to a Histogram bucket.
//
// Must use NewHistogramBucketExemplar* functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type HistogramBucketExemplar struct {
	// Wrap OTLP HistogramDataPoint_Bucket_Exemplar.
	orig *otlpmetrics.HistogramDataPoint_Bucket_Exemplar

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *intenalHistogramBucketExemplar
}

type intenalHistogramBucketExemplar struct {
	attachments Labels
	// True if the pimpl was initialized.
	initialized bool
}

// NewHistogramBucketExemplar creates a new HistogramBucketExemplar.
func NewHistogramBucketExemplar() HistogramBucketExemplar {
	return HistogramBucketExemplar{&otlpmetrics.HistogramDataPoint_Bucket_Exemplar{}, &intenalHistogramBucketExemplar{}}
}

func newHistogramBucketExemplarFromOrig(orig *otlpmetrics.HistogramDataPoint_Bucket_Exemplar) HistogramBucketExemplar {
	return HistogramBucketExemplar{orig, &intenalHistogramBucketExemplar{}}
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
	hbe.initInternallIfNeeded()
	return hbe.pimpl.attachments
}

func (hbe HistogramBucketExemplar) SetAttachments(v Labels) {
	hbe.initInternallIfNeeded()
	hbe.pimpl.attachments = v
}

func (hbe HistogramBucketExemplar) initInternallIfNeeded() {
	if !hbe.pimpl.initialized {
		hbe.pimpl.attachments = newLabelsFromOrig(hbe.orig.Attachments)
		hbe.pimpl.initialized = true
	}
}

func (hbe HistogramBucketExemplar) getOrig() *otlpmetrics.HistogramDataPoint_Bucket_Exemplar {
	return hbe.orig
}

func (hbe HistogramBucketExemplar) flushInternal() {
	if !hbe.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	hbe.orig.Attachments = hbe.pimpl.attachments.toOrig()
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
	labels             Labels
	valueAtPercentiles []SummaryValueAtPercentile
	// True when the slice was replace.
	sliceChanged bool
	// True if the pimpl was initialized.
	initialized bool
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

func (dp SummaryDataPoint) Labels() Labels {
	dp.initInternallIfNeeded()
	return dp.pimpl.labels
}

func (dp SummaryDataPoint) SetLabels(v Labels) {
	dp.initInternallIfNeeded()
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
	dp.initInternallIfNeeded()
	return dp.pimpl.valueAtPercentiles
}

func (dp SummaryDataPoint) SetValueAtPercentiles(v []SummaryValueAtPercentile) {
	dp.initInternallIfNeeded()
	dp.pimpl.valueAtPercentiles = v
	dp.pimpl.sliceChanged = true
}

func (dp SummaryDataPoint) initInternallIfNeeded() {
	if !dp.pimpl.initialized {
		dp.pimpl.labels = newLabelsFromOrig(dp.orig.Labels)
		dp.pimpl.valueAtPercentiles = newSummaryValueAtPercentileSliceFromOrig(dp.orig.PercentileValues)
		dp.pimpl.initialized = true
	}
}

func (dp SummaryDataPoint) getOrig() *otlpmetrics.SummaryDataPoint {
	return dp.orig
}

func (dp SummaryDataPoint) flushInternal() {
	if !dp.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return
	}
	dp.orig.Labels = dp.pimpl.labels.toOrig()
	if dp.pimpl.sliceChanged {
		// Reconstruct the slice because we don't know what elements were removed/added.
		// No internal fields in SummaryValueAtPercentile so no need to flush
		dp.orig.PercentileValues = make([]*otlpmetrics.SummaryDataPoint_ValueAtPercentile, len(dp.pimpl.valueAtPercentiles))
		for i := range dp.pimpl.valueAtPercentiles {
			dp.orig.PercentileValues[i] = dp.pimpl.valueAtPercentiles[i].getOrig()
		}
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

func (vp SummaryValueAtPercentile) getOrig() *otlpmetrics.SummaryDataPoint_ValueAtPercentile {
	return vp.orig
}

// Labels stores the original representation of the labels and the internal LabelsMap representation.
type Labels struct {
	orig []*otlpcommon.StringKeyValue

	// Override a few fields. These fields are the source of truth. Their counterparts
	// stored in corresponding fields of "orig" are ignored.
	pimpl *internalLabel
}

type internalLabel struct {
	// Cannot track changes in the map, so if this is initialized we
	// always reconstruct the labels.
	labelsMap LabelsMap
	// True if the pimpl was initialized.
	initialized bool
}

// NewLabels creates a new Labels.
func NewLabels() Labels {
	return Labels{nil, &internalLabel{}}
}

func newLabelsFromOrig(orig []*otlpcommon.StringKeyValue) Labels {
	return Labels{orig, &internalLabel{}}
}

func (ls Labels) LabelsMap() LabelsMap {
	ls.initInternallIfNeeded()
	return ls.pimpl.labelsMap
}

func (ls Labels) SetLabelsMap(v LabelsMap) {
	ls.initInternallIfNeeded()
	ls.pimpl.labelsMap = v
}

func (ls Labels) initInternallIfNeeded() {
	if !ls.pimpl.initialized {
		if len(ls.orig) == 0 {
			ls.pimpl.labelsMap = LabelsMap{}
			ls.pimpl.initialized = true
			return
		}
		// Extra overhead here if we decode the orig attributes
		// then immediately overwrite them in set.
		labels := make(LabelsMap, len(ls.orig))
		for i := range ls.orig {
			labels[ls.orig[i].GetKey()] = ls.orig[i].GetValue()
		}
		ls.pimpl.labelsMap = labels
		ls.pimpl.initialized = true
	}
}

func (ls Labels) toOrig() []*otlpcommon.StringKeyValue {
	if !ls.pimpl.initialized {
		// Guaranteed no changes via internal fields.
		return ls.orig
	}

	if len(ls.orig) != len(ls.pimpl.labelsMap) {
		skvs := make([]otlpcommon.StringKeyValue, len(ls.pimpl.labelsMap))
		ls.orig = make([]*otlpcommon.StringKeyValue, len(ls.pimpl.labelsMap))
		for i := range ls.orig {
			ls.orig[i] = &skvs[i]
		}
	}

	i := 0
	for k, v := range ls.pimpl.labelsMap {
		ls.orig[i].Key = k
		ls.orig[i].Value = v
		i++
	}

	return ls.orig
}

// LabelsMap stores a map of label keys to values.
type LabelsMap map[string]string
