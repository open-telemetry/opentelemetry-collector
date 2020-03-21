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

// Code generated by "internal/data_generator/main.go". DO NOT EDIT.
// To regenerate this file run "go run internal/data_generator/main.go".

package data

import (
	"testing"
	
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	"github.com/stretchr/testify/assert"
)

func TestResourceMetricsSlice(t *testing.T) {
	es := NewResourceMetricsSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newResourceMetricsSlice(&[]*otlpmetrics.ResourceMetrics{})
	assert.EqualValues(t, 0, es.Len())
	es = NewResourceMetricsSlice(13)
	defaultVal := NewResourceMetrics()
	testVal := generateTestResourceMetrics()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestResourceMetrics(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.ResourceMetrics]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.ResourceMetrics]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.ResourceMetrics]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestResourceMetrics(t *testing.T) {
	ms := NewResourceMetrics()
	assert.EqualValues(t, newResourceMetrics(&otlpmetrics.ResourceMetrics{}), ms)

	assert.EqualValues(t, NewResource(), ms.Resource())
	testValResource := generateTestResource()
	ms.SetResource(testValResource)
	assert.EqualValues(t, testValResource, ms.Resource())

	assert.EqualValues(t, NewInstrumentationLibraryMetricsSlice(0), ms.InstrumentationLibraryMetrics())
	testValInstrumentationLibraryMetrics := generateTestInstrumentationLibraryMetricsSlice()
	ms.SetInstrumentationLibraryMetrics(testValInstrumentationLibraryMetrics)
	assert.EqualValues(t, testValInstrumentationLibraryMetrics, ms.InstrumentationLibraryMetrics())

	assert.EqualValues(t, generateTestResourceMetrics(), ms)
}

func TestInstrumentationLibraryMetricsSlice(t *testing.T) {
	es := NewInstrumentationLibraryMetricsSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newInstrumentationLibraryMetricsSlice(&[]*otlpmetrics.InstrumentationLibraryMetrics{})
	assert.EqualValues(t, 0, es.Len())
	es = NewInstrumentationLibraryMetricsSlice(13)
	defaultVal := NewInstrumentationLibraryMetrics()
	testVal := generateTestInstrumentationLibraryMetrics()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestInstrumentationLibraryMetrics(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestInstrumentationLibraryMetrics(t *testing.T) {
	ms := NewInstrumentationLibraryMetrics()
	assert.EqualValues(t, newInstrumentationLibraryMetrics(&otlpmetrics.InstrumentationLibraryMetrics{}), ms)

	assert.EqualValues(t, NewInstrumentationLibrary(), ms.InstrumentationLibrary())
	testValInstrumentationLibrary := generateTestInstrumentationLibrary()
	ms.SetInstrumentationLibrary(testValInstrumentationLibrary)
	assert.EqualValues(t, testValInstrumentationLibrary, ms.InstrumentationLibrary())

	assert.EqualValues(t, NewMetricSlice(0), ms.Metrics())
	testValMetrics := generateTestMetricSlice()
	ms.SetMetrics(testValMetrics)
	assert.EqualValues(t, testValMetrics, ms.Metrics())

	assert.EqualValues(t, generateTestInstrumentationLibraryMetrics(), ms)
}

func TestMetricSlice(t *testing.T) {
	es := NewMetricSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newMetricSlice(&[]*otlpmetrics.Metric{})
	assert.EqualValues(t, 0, es.Len())
	es = NewMetricSlice(13)
	defaultVal := NewMetric()
	testVal := generateTestMetric()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestMetric(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.Metric]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.Metric]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.Metric]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestMetric(t *testing.T) {
	ms := NewMetric()
	assert.EqualValues(t, newMetric(&otlpmetrics.Metric{}), ms)

	assert.EqualValues(t, NewMetricDescriptor(), ms.MetricDescriptor())
	testValMetricDescriptor := generateTestMetricDescriptor()
	ms.SetMetricDescriptor(testValMetricDescriptor)
	assert.EqualValues(t, testValMetricDescriptor, ms.MetricDescriptor())

	assert.EqualValues(t, NewInt64DataPointSlice(0), ms.Int64DataPoints())
	testValInt64DataPoints := generateTestInt64DataPointSlice()
	ms.SetInt64DataPoints(testValInt64DataPoints)
	assert.EqualValues(t, testValInt64DataPoints, ms.Int64DataPoints())

	assert.EqualValues(t, NewDoubleDataPointSlice(0), ms.DoubleDataPoints())
	testValDoubleDataPoints := generateTestDoubleDataPointSlice()
	ms.SetDoubleDataPoints(testValDoubleDataPoints)
	assert.EqualValues(t, testValDoubleDataPoints, ms.DoubleDataPoints())

	assert.EqualValues(t, NewHistogramDataPointSlice(0), ms.HistogramDataPoints())
	testValHistogramDataPoints := generateTestHistogramDataPointSlice()
	ms.SetHistogramDataPoints(testValHistogramDataPoints)
	assert.EqualValues(t, testValHistogramDataPoints, ms.HistogramDataPoints())

	assert.EqualValues(t, NewSummaryDataPointSlice(0), ms.SummaryDataPoints())
	testValSummaryDataPoints := generateTestSummaryDataPointSlice()
	ms.SetSummaryDataPoints(testValSummaryDataPoints)
	assert.EqualValues(t, testValSummaryDataPoints, ms.SummaryDataPoints())

	assert.EqualValues(t, generateTestMetric(), ms)
}

func TestMetricDescriptor(t *testing.T) {
	ms := NewMetricDescriptor()
	assert.EqualValues(t, newMetricDescriptor(&otlpmetrics.MetricDescriptor{}), ms)

	assert.EqualValues(t, "", ms.Name())
	testValName := "test_name"
	ms.SetName(testValName)
	assert.EqualValues(t, testValName, ms.Name())

	assert.EqualValues(t, "", ms.Description())
	testValDescription := "test_description"
	ms.SetDescription(testValDescription)
	assert.EqualValues(t, testValDescription, ms.Description())

	assert.EqualValues(t, "", ms.Unit())
	testValUnit := "1"
	ms.SetUnit(testValUnit)
	assert.EqualValues(t, testValUnit, ms.Unit())

	assert.EqualValues(t, MetricTypeUnspecified, ms.Type())
	testValType := MetricTypeGaugeInt64
	ms.SetType(testValType)
	assert.EqualValues(t, testValType, ms.Type())

	assert.EqualValues(t, NewStringMap(nil), ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	ms.SetLabelsMap(testValLabelsMap)
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())

	assert.EqualValues(t, generateTestMetricDescriptor(), ms)
}

func TestInt64DataPointSlice(t *testing.T) {
	es := NewInt64DataPointSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newInt64DataPointSlice(&[]*otlpmetrics.Int64DataPoint{})
	assert.EqualValues(t, 0, es.Len())
	es = NewInt64DataPointSlice(13)
	defaultVal := NewInt64DataPoint()
	testVal := generateTestInt64DataPoint()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestInt64DataPoint(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.Int64DataPoint]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.Int64DataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.Int64DataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestInt64DataPoint(t *testing.T) {
	ms := NewInt64DataPoint()
	assert.EqualValues(t, newInt64DataPoint(&otlpmetrics.Int64DataPoint{}), ms)

	assert.EqualValues(t, NewStringMap(nil), ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	ms.SetLabelsMap(testValLabelsMap)
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())

	assert.EqualValues(t, TimestampUnixNano(0), ms.StartTime())
	testValStartTime := TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())

	assert.EqualValues(t, TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())

	assert.EqualValues(t, int64(0), ms.Value())
	testValValue := int64(-17)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())

	assert.EqualValues(t, generateTestInt64DataPoint(), ms)
}

func TestDoubleDataPointSlice(t *testing.T) {
	es := NewDoubleDataPointSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newDoubleDataPointSlice(&[]*otlpmetrics.DoubleDataPoint{})
	assert.EqualValues(t, 0, es.Len())
	es = NewDoubleDataPointSlice(13)
	defaultVal := NewDoubleDataPoint()
	testVal := generateTestDoubleDataPoint()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestDoubleDataPoint(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.DoubleDataPoint]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.DoubleDataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.DoubleDataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestDoubleDataPoint(t *testing.T) {
	ms := NewDoubleDataPoint()
	assert.EqualValues(t, newDoubleDataPoint(&otlpmetrics.DoubleDataPoint{}), ms)

	assert.EqualValues(t, NewStringMap(nil), ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	ms.SetLabelsMap(testValLabelsMap)
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())

	assert.EqualValues(t, TimestampUnixNano(0), ms.StartTime())
	testValStartTime := TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())

	assert.EqualValues(t, TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())

	assert.EqualValues(t, float64(0.0), ms.Value())
	testValValue := float64(17.13)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())

	assert.EqualValues(t, generateTestDoubleDataPoint(), ms)
}

func TestHistogramDataPointSlice(t *testing.T) {
	es := NewHistogramDataPointSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newHistogramDataPointSlice(&[]*otlpmetrics.HistogramDataPoint{})
	assert.EqualValues(t, 0, es.Len())
	es = NewHistogramDataPointSlice(13)
	defaultVal := NewHistogramDataPoint()
	testVal := generateTestHistogramDataPoint()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestHistogramDataPoint(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.HistogramDataPoint]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.HistogramDataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.HistogramDataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestHistogramDataPoint(t *testing.T) {
	ms := NewHistogramDataPoint()
	assert.EqualValues(t, newHistogramDataPoint(&otlpmetrics.HistogramDataPoint{}), ms)

	assert.EqualValues(t, NewStringMap(nil), ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	ms.SetLabelsMap(testValLabelsMap)
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())

	assert.EqualValues(t, TimestampUnixNano(0), ms.StartTime())
	testValStartTime := TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())

	assert.EqualValues(t, TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())

	assert.EqualValues(t, uint64(0), ms.Count())
	testValCount := uint64(17)
	ms.SetCount(testValCount)
	assert.EqualValues(t, testValCount, ms.Count())

	assert.EqualValues(t, float64(0.0), ms.Sum())
	testValSum := float64(17.13)
	ms.SetSum(testValSum)
	assert.EqualValues(t, testValSum, ms.Sum())

	assert.EqualValues(t, NewHistogramBucketSlice(0), ms.Buckets())
	testValBuckets := generateTestHistogramBucketSlice()
	ms.SetBuckets(testValBuckets)
	assert.EqualValues(t, testValBuckets, ms.Buckets())

	assert.EqualValues(t, []float64(nil), ms.ExplicitBounds())
	testValExplicitBounds := []float64{1, 2, 3}
	ms.SetExplicitBounds(testValExplicitBounds)
	assert.EqualValues(t, testValExplicitBounds, ms.ExplicitBounds())

	assert.EqualValues(t, generateTestHistogramDataPoint(), ms)
}

func TestHistogramBucketSlice(t *testing.T) {
	es := NewHistogramBucketSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newHistogramBucketSlice(&[]*otlpmetrics.HistogramDataPoint_Bucket{})
	assert.EqualValues(t, 0, es.Len())
	es = NewHistogramBucketSlice(13)
	defaultVal := NewHistogramBucket()
	testVal := generateTestHistogramBucket()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestHistogramBucket(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestHistogramBucket(t *testing.T) {
	ms := NewHistogramBucket()
	assert.EqualValues(t, newHistogramBucket(&otlpmetrics.HistogramDataPoint_Bucket{}), ms)

	assert.EqualValues(t, uint64(0), ms.Count())
	testValCount := uint64(17)
	ms.SetCount(testValCount)
	assert.EqualValues(t, testValCount, ms.Count())

	assert.EqualValues(t, NewHistogramBucketExemplar(), ms.Exemplar())
	testValExemplar := generateTestHistogramBucketExemplar()
	ms.SetExemplar(testValExemplar)
	assert.EqualValues(t, testValExemplar, ms.Exemplar())

	assert.EqualValues(t, generateTestHistogramBucket(), ms)
}

func TestHistogramBucketExemplar(t *testing.T) {
	ms := NewHistogramBucketExemplar()
	assert.EqualValues(t, newHistogramBucketExemplar(&otlpmetrics.HistogramDataPoint_Bucket_Exemplar{}), ms)

	assert.EqualValues(t, TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())

	assert.EqualValues(t, float64(0.0), ms.Value())
	testValValue := float64(17.13)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())

	assert.EqualValues(t, NewStringMap(nil), ms.Attachments())
	testValAttachments := generateTestStringMap()
	ms.SetAttachments(testValAttachments)
	assert.EqualValues(t, testValAttachments, ms.Attachments())

	assert.EqualValues(t, generateTestHistogramBucketExemplar(), ms)
}

func TestSummaryDataPointSlice(t *testing.T) {
	es := NewSummaryDataPointSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newSummaryDataPointSlice(&[]*otlpmetrics.SummaryDataPoint{})
	assert.EqualValues(t, 0, es.Len())
	es = NewSummaryDataPointSlice(13)
	defaultVal := NewSummaryDataPoint()
	testVal := generateTestSummaryDataPoint()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestSummaryDataPoint(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.SummaryDataPoint]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.SummaryDataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.SummaryDataPoint]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestSummaryDataPoint(t *testing.T) {
	ms := NewSummaryDataPoint()
	assert.EqualValues(t, newSummaryDataPoint(&otlpmetrics.SummaryDataPoint{}), ms)

	assert.EqualValues(t, NewStringMap(nil), ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	ms.SetLabelsMap(testValLabelsMap)
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())

	assert.EqualValues(t, TimestampUnixNano(0), ms.StartTime())
	testValStartTime := TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())

	assert.EqualValues(t, TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())

	assert.EqualValues(t, uint64(0), ms.Count())
	testValCount := uint64(17)
	ms.SetCount(testValCount)
	assert.EqualValues(t, testValCount, ms.Count())

	assert.EqualValues(t, float64(0.0), ms.Sum())
	testValSum := float64(17.13)
	ms.SetSum(testValSum)
	assert.EqualValues(t, testValSum, ms.Sum())

	assert.EqualValues(t, NewSummaryValueAtPercentileSlice(0), ms.ValueAtPercentiles())
	testValValueAtPercentiles := generateTestSummaryValueAtPercentileSlice()
	ms.SetValueAtPercentiles(testValValueAtPercentiles)
	assert.EqualValues(t, testValValueAtPercentiles, ms.ValueAtPercentiles())

	assert.EqualValues(t, generateTestSummaryDataPoint(), ms)
}

func TestSummaryValueAtPercentileSlice(t *testing.T) {
	es := NewSummaryValueAtPercentileSlice(0)
	assert.EqualValues(t, 0, es.Len())
	es = newSummaryValueAtPercentileSlice(&[]*otlpmetrics.SummaryDataPoint_ValueAtPercentile{})
	assert.EqualValues(t, 0, es.Len())
	es = NewSummaryValueAtPercentileSlice(13)
	defaultVal := NewSummaryValueAtPercentile()
	testVal := generateTestSummaryValueAtPercentile()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTestSummaryValueAtPercentile(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}

func TestSummaryValueAtPercentile(t *testing.T) {
	ms := NewSummaryValueAtPercentile()
	assert.EqualValues(t, newSummaryValueAtPercentile(&otlpmetrics.SummaryDataPoint_ValueAtPercentile{}), ms)

	assert.EqualValues(t, float64(0.0), ms.Percentile())
	testValPercentile := float64(0.90)
	ms.SetPercentile(testValPercentile)
	assert.EqualValues(t, testValPercentile, ms.Percentile())

	assert.EqualValues(t, float64(0.0), ms.Value())
	testValValue := float64(17.13)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())

	assert.EqualValues(t, generateTestSummaryValueAtPercentile(), ms)
}

func generateTestResourceMetricsSlice() ResourceMetricsSlice {
	tv := NewResourceMetricsSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestResourceMetrics(tv.Get(i))
	}
	return tv
}

func generateTestResourceMetrics() ResourceMetrics {
	tv := NewResourceMetrics()
	tv.SetResource(generateTestResource())
	tv.SetInstrumentationLibraryMetrics(generateTestInstrumentationLibraryMetricsSlice())
	return tv
}

func fillTestResourceMetrics(tv ResourceMetrics) {
	tv.SetResource(generateTestResource())
	tv.SetInstrumentationLibraryMetrics(generateTestInstrumentationLibraryMetricsSlice())
}

func generateTestInstrumentationLibraryMetricsSlice() InstrumentationLibraryMetricsSlice {
	tv := NewInstrumentationLibraryMetricsSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestInstrumentationLibraryMetrics(tv.Get(i))
	}
	return tv
}

func generateTestInstrumentationLibraryMetrics() InstrumentationLibraryMetrics {
	tv := NewInstrumentationLibraryMetrics()
	tv.SetInstrumentationLibrary(generateTestInstrumentationLibrary())
	tv.SetMetrics(generateTestMetricSlice())
	return tv
}

func fillTestInstrumentationLibraryMetrics(tv InstrumentationLibraryMetrics) {
	tv.SetInstrumentationLibrary(generateTestInstrumentationLibrary())
	tv.SetMetrics(generateTestMetricSlice())
}

func generateTestMetricSlice() MetricSlice {
	tv := NewMetricSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestMetric(tv.Get(i))
	}
	return tv
}

func generateTestMetric() Metric {
	tv := NewMetric()
	tv.SetMetricDescriptor(generateTestMetricDescriptor())
	tv.SetInt64DataPoints(generateTestInt64DataPointSlice())
	tv.SetDoubleDataPoints(generateTestDoubleDataPointSlice())
	tv.SetHistogramDataPoints(generateTestHistogramDataPointSlice())
	tv.SetSummaryDataPoints(generateTestSummaryDataPointSlice())
	return tv
}

func fillTestMetric(tv Metric) {
	tv.SetMetricDescriptor(generateTestMetricDescriptor())
	tv.SetInt64DataPoints(generateTestInt64DataPointSlice())
	tv.SetDoubleDataPoints(generateTestDoubleDataPointSlice())
	tv.SetHistogramDataPoints(generateTestHistogramDataPointSlice())
	tv.SetSummaryDataPoints(generateTestSummaryDataPointSlice())
}

func generateTestMetricDescriptor() MetricDescriptor {
	tv := NewMetricDescriptor()
	tv.SetName("test_name")
	tv.SetDescription("test_description")
	tv.SetUnit("1")
	tv.SetType(MetricTypeGaugeInt64)
	tv.SetLabelsMap(generateTestStringMap())
	return tv
}

func generateTestInt64DataPointSlice() Int64DataPointSlice {
	tv := NewInt64DataPointSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestInt64DataPoint(tv.Get(i))
	}
	return tv
}

func generateTestInt64DataPoint() Int64DataPoint {
	tv := NewInt64DataPoint()
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetValue(int64(-17))
	return tv
}

func fillTestInt64DataPoint(tv Int64DataPoint) {
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetValue(int64(-17))
}

func generateTestDoubleDataPointSlice() DoubleDataPointSlice {
	tv := NewDoubleDataPointSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestDoubleDataPoint(tv.Get(i))
	}
	return tv
}

func generateTestDoubleDataPoint() DoubleDataPoint {
	tv := NewDoubleDataPoint()
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetValue(float64(17.13))
	return tv
}

func fillTestDoubleDataPoint(tv DoubleDataPoint) {
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetValue(float64(17.13))
}

func generateTestHistogramDataPointSlice() HistogramDataPointSlice {
	tv := NewHistogramDataPointSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestHistogramDataPoint(tv.Get(i))
	}
	return tv
}

func generateTestHistogramDataPoint() HistogramDataPoint {
	tv := NewHistogramDataPoint()
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetCount(uint64(17))
	tv.SetSum(float64(17.13))
	tv.SetBuckets(generateTestHistogramBucketSlice())
	tv.SetExplicitBounds([]float64{1, 2, 3})
	return tv
}

func fillTestHistogramDataPoint(tv HistogramDataPoint) {
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetCount(uint64(17))
	tv.SetSum(float64(17.13))
	tv.SetBuckets(generateTestHistogramBucketSlice())
	tv.SetExplicitBounds([]float64{1, 2, 3})
}

func generateTestHistogramBucketSlice() HistogramBucketSlice {
	tv := NewHistogramBucketSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestHistogramBucket(tv.Get(i))
	}
	return tv
}

func generateTestHistogramBucket() HistogramBucket {
	tv := NewHistogramBucket()
	tv.SetCount(uint64(17))
	tv.SetExemplar(generateTestHistogramBucketExemplar())
	return tv
}

func fillTestHistogramBucket(tv HistogramBucket) {
	tv.SetCount(uint64(17))
	tv.SetExemplar(generateTestHistogramBucketExemplar())
}

func generateTestHistogramBucketExemplar() HistogramBucketExemplar {
	tv := NewHistogramBucketExemplar()
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetValue(float64(17.13))
	tv.SetAttachments(generateTestStringMap())
	return tv
}

func generateTestSummaryDataPointSlice() SummaryDataPointSlice {
	tv := NewSummaryDataPointSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestSummaryDataPoint(tv.Get(i))
	}
	return tv
}

func generateTestSummaryDataPoint() SummaryDataPoint {
	tv := NewSummaryDataPoint()
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetCount(uint64(17))
	tv.SetSum(float64(17.13))
	tv.SetValueAtPercentiles(generateTestSummaryValueAtPercentileSlice())
	return tv
}

func fillTestSummaryDataPoint(tv SummaryDataPoint) {
	tv.SetLabelsMap(generateTestStringMap())
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetCount(uint64(17))
	tv.SetSum(float64(17.13))
	tv.SetValueAtPercentiles(generateTestSummaryValueAtPercentileSlice())
}

func generateTestSummaryValueAtPercentileSlice() SummaryValueAtPercentileSlice {
	tv := NewSummaryValueAtPercentileSlice(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestSummaryValueAtPercentile(tv.Get(i))
	}
	return tv
}

func generateTestSummaryValueAtPercentile() SummaryValueAtPercentile {
	tv := NewSummaryValueAtPercentile()
	tv.SetPercentile(float64(0.90))
	tv.SetValue(float64(17.13))
	return tv
}

func fillTestSummaryValueAtPercentile(tv SummaryValueAtPercentile) {
	tv.SetPercentile(float64(0.90))
	tv.SetValue(float64(17.13))
}
