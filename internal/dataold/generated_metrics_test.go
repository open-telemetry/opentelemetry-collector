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

package dataold

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1old"
)

func TestResourceMetricsSlice(t *testing.T) {
	es := NewResourceMetricsSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newResourceMetricsSlice(&[]*otlpmetrics.ResourceMetrics{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewResourceMetrics()
	emptyVal.InitEmpty()
	testVal := generateTestResourceMetrics()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestResourceMetrics(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestResourceMetricsSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestResourceMetricsSlice()
	dest := NewResourceMetricsSlice()
	src := generateTestResourceMetricsSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestResourceMetricsSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestResourceMetricsSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestResourceMetricsSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestResourceMetricsSlice_CopyTo(t *testing.T) {
	dest := NewResourceMetricsSlice()
	// Test CopyTo to empty
	NewResourceMetricsSlice().CopyTo(dest)
	assert.EqualValues(t, NewResourceMetricsSlice(), dest)

	// Test CopyTo larger slice
	generateTestResourceMetricsSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestResourceMetricsSlice(), dest)

	// Test CopyTo same size slice
	generateTestResourceMetricsSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestResourceMetricsSlice(), dest)
}

func TestResourceMetricsSlice_Resize(t *testing.T) {
	es := generateTestResourceMetricsSlice()
	emptyVal := NewResourceMetrics()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.ResourceMetrics]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.ResourceMetrics]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.ResourceMetrics]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.ResourceMetrics]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewResourceMetricsSlice(), es)
}

func TestResourceMetricsSlice_Append(t *testing.T) {
	es := generateTestResourceMetricsSlice()
	emptyVal := NewResourceMetrics()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewResourceMetrics()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestResourceMetrics_InitEmpty(t *testing.T) {
	ms := NewResourceMetrics()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestResourceMetrics_CopyTo(t *testing.T) {
	ms := NewResourceMetrics()
	NewResourceMetrics().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestResourceMetrics().CopyTo(ms)
	assert.EqualValues(t, generateTestResourceMetrics(), ms)
}

func TestResourceMetrics_Resource(t *testing.T) {
	ms := NewResourceMetrics()
	ms.InitEmpty()
	assert.EqualValues(t, true, ms.Resource().IsNil())
	ms.Resource().InitEmpty()
	assert.EqualValues(t, false, ms.Resource().IsNil())
	fillTestResource(ms.Resource())
	assert.EqualValues(t, generateTestResource(), ms.Resource())
}

func TestResourceMetrics_InstrumentationLibraryMetrics(t *testing.T) {
	ms := NewResourceMetrics()
	ms.InitEmpty()
	assert.EqualValues(t, NewInstrumentationLibraryMetricsSlice(), ms.InstrumentationLibraryMetrics())
	fillTestInstrumentationLibraryMetricsSlice(ms.InstrumentationLibraryMetrics())
	testValInstrumentationLibraryMetrics := generateTestInstrumentationLibraryMetricsSlice()
	assert.EqualValues(t, testValInstrumentationLibraryMetrics, ms.InstrumentationLibraryMetrics())
}

func TestInstrumentationLibraryMetricsSlice(t *testing.T) {
	es := NewInstrumentationLibraryMetricsSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newInstrumentationLibraryMetricsSlice(&[]*otlpmetrics.InstrumentationLibraryMetrics{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewInstrumentationLibraryMetrics()
	emptyVal.InitEmpty()
	testVal := generateTestInstrumentationLibraryMetrics()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestInstrumentationLibraryMetrics(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestInstrumentationLibraryMetricsSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestInstrumentationLibraryMetricsSlice()
	dest := NewInstrumentationLibraryMetricsSlice()
	src := generateTestInstrumentationLibraryMetricsSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestInstrumentationLibraryMetricsSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestInstrumentationLibraryMetricsSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestInstrumentationLibraryMetricsSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestInstrumentationLibraryMetricsSlice_CopyTo(t *testing.T) {
	dest := NewInstrumentationLibraryMetricsSlice()
	// Test CopyTo to empty
	NewInstrumentationLibraryMetricsSlice().CopyTo(dest)
	assert.EqualValues(t, NewInstrumentationLibraryMetricsSlice(), dest)

	// Test CopyTo larger slice
	generateTestInstrumentationLibraryMetricsSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestInstrumentationLibraryMetricsSlice(), dest)

	// Test CopyTo same size slice
	generateTestInstrumentationLibraryMetricsSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestInstrumentationLibraryMetricsSlice(), dest)
}

func TestInstrumentationLibraryMetricsSlice_Resize(t *testing.T) {
	es := generateTestInstrumentationLibraryMetricsSlice()
	emptyVal := NewInstrumentationLibraryMetrics()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.InstrumentationLibraryMetrics]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewInstrumentationLibraryMetricsSlice(), es)
}

func TestInstrumentationLibraryMetricsSlice_Append(t *testing.T) {
	es := generateTestInstrumentationLibraryMetricsSlice()
	emptyVal := NewInstrumentationLibraryMetrics()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewInstrumentationLibraryMetrics()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestInstrumentationLibraryMetrics_InitEmpty(t *testing.T) {
	ms := NewInstrumentationLibraryMetrics()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestInstrumentationLibraryMetrics_CopyTo(t *testing.T) {
	ms := NewInstrumentationLibraryMetrics()
	NewInstrumentationLibraryMetrics().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestInstrumentationLibraryMetrics().CopyTo(ms)
	assert.EqualValues(t, generateTestInstrumentationLibraryMetrics(), ms)
}

func TestInstrumentationLibraryMetrics_InstrumentationLibrary(t *testing.T) {
	ms := NewInstrumentationLibraryMetrics()
	ms.InitEmpty()
	assert.EqualValues(t, true, ms.InstrumentationLibrary().IsNil())
	ms.InstrumentationLibrary().InitEmpty()
	assert.EqualValues(t, false, ms.InstrumentationLibrary().IsNil())
	fillTestInstrumentationLibrary(ms.InstrumentationLibrary())
	assert.EqualValues(t, generateTestInstrumentationLibrary(), ms.InstrumentationLibrary())
}

func TestInstrumentationLibraryMetrics_Metrics(t *testing.T) {
	ms := NewInstrumentationLibraryMetrics()
	ms.InitEmpty()
	assert.EqualValues(t, NewMetricSlice(), ms.Metrics())
	fillTestMetricSlice(ms.Metrics())
	testValMetrics := generateTestMetricSlice()
	assert.EqualValues(t, testValMetrics, ms.Metrics())
}

func TestMetricSlice(t *testing.T) {
	es := NewMetricSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newMetricSlice(&[]*otlpmetrics.Metric{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewMetric()
	emptyVal.InitEmpty()
	testVal := generateTestMetric()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestMetric(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestMetricSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestMetricSlice()
	dest := NewMetricSlice()
	src := generateTestMetricSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestMetricSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestMetricSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestMetricSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestMetricSlice_CopyTo(t *testing.T) {
	dest := NewMetricSlice()
	// Test CopyTo to empty
	NewMetricSlice().CopyTo(dest)
	assert.EqualValues(t, NewMetricSlice(), dest)

	// Test CopyTo larger slice
	generateTestMetricSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestMetricSlice(), dest)

	// Test CopyTo same size slice
	generateTestMetricSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestMetricSlice(), dest)
}

func TestMetricSlice_Resize(t *testing.T) {
	es := generateTestMetricSlice()
	emptyVal := NewMetric()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.Metric]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.Metric]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.Metric]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.Metric]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewMetricSlice(), es)
}

func TestMetricSlice_Append(t *testing.T) {
	es := generateTestMetricSlice()
	emptyVal := NewMetric()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewMetric()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestMetric_InitEmpty(t *testing.T) {
	ms := NewMetric()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestMetric_CopyTo(t *testing.T) {
	ms := NewMetric()
	NewMetric().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestMetric().CopyTo(ms)
	assert.EqualValues(t, generateTestMetric(), ms)
}

func TestMetric_MetricDescriptor(t *testing.T) {
	ms := NewMetric()
	ms.InitEmpty()
	assert.EqualValues(t, true, ms.MetricDescriptor().IsNil())
	ms.MetricDescriptor().InitEmpty()
	assert.EqualValues(t, false, ms.MetricDescriptor().IsNil())
	fillTestMetricDescriptor(ms.MetricDescriptor())
	assert.EqualValues(t, generateTestMetricDescriptor(), ms.MetricDescriptor())
}

func TestMetric_Int64DataPoints(t *testing.T) {
	ms := NewMetric()
	ms.InitEmpty()
	assert.EqualValues(t, NewInt64DataPointSlice(), ms.Int64DataPoints())
	fillTestInt64DataPointSlice(ms.Int64DataPoints())
	testValInt64DataPoints := generateTestInt64DataPointSlice()
	assert.EqualValues(t, testValInt64DataPoints, ms.Int64DataPoints())
}

func TestMetric_DoubleDataPoints(t *testing.T) {
	ms := NewMetric()
	ms.InitEmpty()
	assert.EqualValues(t, NewDoubleDataPointSlice(), ms.DoubleDataPoints())
	fillTestDoubleDataPointSlice(ms.DoubleDataPoints())
	testValDoubleDataPoints := generateTestDoubleDataPointSlice()
	assert.EqualValues(t, testValDoubleDataPoints, ms.DoubleDataPoints())
}

func TestMetric_HistogramDataPoints(t *testing.T) {
	ms := NewMetric()
	ms.InitEmpty()
	assert.EqualValues(t, NewHistogramDataPointSlice(), ms.HistogramDataPoints())
	fillTestHistogramDataPointSlice(ms.HistogramDataPoints())
	testValHistogramDataPoints := generateTestHistogramDataPointSlice()
	assert.EqualValues(t, testValHistogramDataPoints, ms.HistogramDataPoints())
}

func TestMetric_SummaryDataPoints(t *testing.T) {
	ms := NewMetric()
	ms.InitEmpty()
	assert.EqualValues(t, NewSummaryDataPointSlice(), ms.SummaryDataPoints())
	fillTestSummaryDataPointSlice(ms.SummaryDataPoints())
	testValSummaryDataPoints := generateTestSummaryDataPointSlice()
	assert.EqualValues(t, testValSummaryDataPoints, ms.SummaryDataPoints())
}

func TestMetricDescriptor_InitEmpty(t *testing.T) {
	ms := NewMetricDescriptor()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestMetricDescriptor_CopyTo(t *testing.T) {
	ms := NewMetricDescriptor()
	NewMetricDescriptor().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestMetricDescriptor().CopyTo(ms)
	assert.EqualValues(t, generateTestMetricDescriptor(), ms)
}

func TestMetricDescriptor_Name(t *testing.T) {
	ms := NewMetricDescriptor()
	ms.InitEmpty()
	assert.EqualValues(t, "", ms.Name())
	testValName := "test_name"
	ms.SetName(testValName)
	assert.EqualValues(t, testValName, ms.Name())
}

func TestMetricDescriptor_Description(t *testing.T) {
	ms := NewMetricDescriptor()
	ms.InitEmpty()
	assert.EqualValues(t, "", ms.Description())
	testValDescription := "test_description"
	ms.SetDescription(testValDescription)
	assert.EqualValues(t, testValDescription, ms.Description())
}

func TestMetricDescriptor_Unit(t *testing.T) {
	ms := NewMetricDescriptor()
	ms.InitEmpty()
	assert.EqualValues(t, "", ms.Unit())
	testValUnit := "1"
	ms.SetUnit(testValUnit)
	assert.EqualValues(t, testValUnit, ms.Unit())
}

func TestMetricDescriptor_Type(t *testing.T) {
	ms := NewMetricDescriptor()
	ms.InitEmpty()
	assert.EqualValues(t, MetricTypeInvalid, ms.Type())
	testValType := MetricTypeInt64
	ms.SetType(testValType)
	assert.EqualValues(t, testValType, ms.Type())
}

func TestInt64DataPointSlice(t *testing.T) {
	es := NewInt64DataPointSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newInt64DataPointSlice(&[]*otlpmetrics.Int64DataPoint{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewInt64DataPoint()
	emptyVal.InitEmpty()
	testVal := generateTestInt64DataPoint()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestInt64DataPoint(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestInt64DataPointSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestInt64DataPointSlice()
	dest := NewInt64DataPointSlice()
	src := generateTestInt64DataPointSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestInt64DataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestInt64DataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestInt64DataPointSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestInt64DataPointSlice_CopyTo(t *testing.T) {
	dest := NewInt64DataPointSlice()
	// Test CopyTo to empty
	NewInt64DataPointSlice().CopyTo(dest)
	assert.EqualValues(t, NewInt64DataPointSlice(), dest)

	// Test CopyTo larger slice
	generateTestInt64DataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestInt64DataPointSlice(), dest)

	// Test CopyTo same size slice
	generateTestInt64DataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestInt64DataPointSlice(), dest)
}

func TestInt64DataPointSlice_Resize(t *testing.T) {
	es := generateTestInt64DataPointSlice()
	emptyVal := NewInt64DataPoint()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.Int64DataPoint]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.Int64DataPoint]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.Int64DataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.Int64DataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewInt64DataPointSlice(), es)
}

func TestInt64DataPointSlice_Append(t *testing.T) {
	es := generateTestInt64DataPointSlice()
	emptyVal := NewInt64DataPoint()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewInt64DataPoint()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestInt64DataPoint_InitEmpty(t *testing.T) {
	ms := NewInt64DataPoint()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestInt64DataPoint_CopyTo(t *testing.T) {
	ms := NewInt64DataPoint()
	NewInt64DataPoint().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestInt64DataPoint().CopyTo(ms)
	assert.EqualValues(t, generateTestInt64DataPoint(), ms)
}

func TestInt64DataPoint_LabelsMap(t *testing.T) {
	ms := NewInt64DataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.NewStringMap(), ms.LabelsMap())
	fillTestStringMap(ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())
}

func TestInt64DataPoint_StartTime(t *testing.T) {
	ms := NewInt64DataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.StartTime())
	testValStartTime := pdata.TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())
}

func TestInt64DataPoint_Timestamp(t *testing.T) {
	ms := NewInt64DataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := pdata.TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())
}

func TestInt64DataPoint_Value(t *testing.T) {
	ms := NewInt64DataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, int64(0), ms.Value())
	testValValue := int64(-17)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())
}

func TestDoubleDataPointSlice(t *testing.T) {
	es := NewDoubleDataPointSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newDoubleDataPointSlice(&[]*otlpmetrics.DoubleDataPoint{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewDoubleDataPoint()
	emptyVal.InitEmpty()
	testVal := generateTestDoubleDataPoint()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestDoubleDataPoint(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestDoubleDataPointSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestDoubleDataPointSlice()
	dest := NewDoubleDataPointSlice()
	src := generateTestDoubleDataPointSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestDoubleDataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestDoubleDataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestDoubleDataPointSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestDoubleDataPointSlice_CopyTo(t *testing.T) {
	dest := NewDoubleDataPointSlice()
	// Test CopyTo to empty
	NewDoubleDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, NewDoubleDataPointSlice(), dest)

	// Test CopyTo larger slice
	generateTestDoubleDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestDoubleDataPointSlice(), dest)

	// Test CopyTo same size slice
	generateTestDoubleDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestDoubleDataPointSlice(), dest)
}

func TestDoubleDataPointSlice_Resize(t *testing.T) {
	es := generateTestDoubleDataPointSlice()
	emptyVal := NewDoubleDataPoint()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.DoubleDataPoint]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.DoubleDataPoint]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.DoubleDataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.DoubleDataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewDoubleDataPointSlice(), es)
}

func TestDoubleDataPointSlice_Append(t *testing.T) {
	es := generateTestDoubleDataPointSlice()
	emptyVal := NewDoubleDataPoint()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewDoubleDataPoint()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestDoubleDataPoint_InitEmpty(t *testing.T) {
	ms := NewDoubleDataPoint()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestDoubleDataPoint_CopyTo(t *testing.T) {
	ms := NewDoubleDataPoint()
	NewDoubleDataPoint().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestDoubleDataPoint().CopyTo(ms)
	assert.EqualValues(t, generateTestDoubleDataPoint(), ms)
}

func TestDoubleDataPoint_LabelsMap(t *testing.T) {
	ms := NewDoubleDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.NewStringMap(), ms.LabelsMap())
	fillTestStringMap(ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())
}

func TestDoubleDataPoint_StartTime(t *testing.T) {
	ms := NewDoubleDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.StartTime())
	testValStartTime := pdata.TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())
}

func TestDoubleDataPoint_Timestamp(t *testing.T) {
	ms := NewDoubleDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := pdata.TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())
}

func TestDoubleDataPoint_Value(t *testing.T) {
	ms := NewDoubleDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, float64(0.0), ms.Value())
	testValValue := float64(17.13)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())
}

func TestHistogramDataPointSlice(t *testing.T) {
	es := NewHistogramDataPointSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newHistogramDataPointSlice(&[]*otlpmetrics.HistogramDataPoint{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewHistogramDataPoint()
	emptyVal.InitEmpty()
	testVal := generateTestHistogramDataPoint()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestHistogramDataPoint(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestHistogramDataPointSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestHistogramDataPointSlice()
	dest := NewHistogramDataPointSlice()
	src := generateTestHistogramDataPointSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestHistogramDataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestHistogramDataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestHistogramDataPointSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestHistogramDataPointSlice_CopyTo(t *testing.T) {
	dest := NewHistogramDataPointSlice()
	// Test CopyTo to empty
	NewHistogramDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, NewHistogramDataPointSlice(), dest)

	// Test CopyTo larger slice
	generateTestHistogramDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestHistogramDataPointSlice(), dest)

	// Test CopyTo same size slice
	generateTestHistogramDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestHistogramDataPointSlice(), dest)
}

func TestHistogramDataPointSlice_Resize(t *testing.T) {
	es := generateTestHistogramDataPointSlice()
	emptyVal := NewHistogramDataPoint()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.HistogramDataPoint]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.HistogramDataPoint]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.HistogramDataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.HistogramDataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewHistogramDataPointSlice(), es)
}

func TestHistogramDataPointSlice_Append(t *testing.T) {
	es := generateTestHistogramDataPointSlice()
	emptyVal := NewHistogramDataPoint()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewHistogramDataPoint()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestHistogramDataPoint_InitEmpty(t *testing.T) {
	ms := NewHistogramDataPoint()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestHistogramDataPoint_CopyTo(t *testing.T) {
	ms := NewHistogramDataPoint()
	NewHistogramDataPoint().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestHistogramDataPoint().CopyTo(ms)
	assert.EqualValues(t, generateTestHistogramDataPoint(), ms)
}

func TestHistogramDataPoint_LabelsMap(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.NewStringMap(), ms.LabelsMap())
	fillTestStringMap(ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())
}

func TestHistogramDataPoint_StartTime(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.StartTime())
	testValStartTime := pdata.TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())
}

func TestHistogramDataPoint_Timestamp(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := pdata.TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())
}

func TestHistogramDataPoint_Count(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, uint64(0), ms.Count())
	testValCount := uint64(17)
	ms.SetCount(testValCount)
	assert.EqualValues(t, testValCount, ms.Count())
}

func TestHistogramDataPoint_Sum(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, float64(0.0), ms.Sum())
	testValSum := float64(17.13)
	ms.SetSum(testValSum)
	assert.EqualValues(t, testValSum, ms.Sum())
}

func TestHistogramDataPoint_Buckets(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, NewHistogramBucketSlice(), ms.Buckets())
	fillTestHistogramBucketSlice(ms.Buckets())
	testValBuckets := generateTestHistogramBucketSlice()
	assert.EqualValues(t, testValBuckets, ms.Buckets())
}

func TestHistogramDataPoint_ExplicitBounds(t *testing.T) {
	ms := NewHistogramDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, []float64(nil), ms.ExplicitBounds())
	testValExplicitBounds := []float64{1, 2, 3}
	ms.SetExplicitBounds(testValExplicitBounds)
	assert.EqualValues(t, testValExplicitBounds, ms.ExplicitBounds())
}

func TestHistogramBucketSlice(t *testing.T) {
	es := NewHistogramBucketSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newHistogramBucketSlice(&[]*otlpmetrics.HistogramDataPoint_Bucket{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewHistogramBucket()
	emptyVal.InitEmpty()
	testVal := generateTestHistogramBucket()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestHistogramBucket(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestHistogramBucketSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestHistogramBucketSlice()
	dest := NewHistogramBucketSlice()
	src := generateTestHistogramBucketSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestHistogramBucketSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestHistogramBucketSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestHistogramBucketSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestHistogramBucketSlice_CopyTo(t *testing.T) {
	dest := NewHistogramBucketSlice()
	// Test CopyTo to empty
	NewHistogramBucketSlice().CopyTo(dest)
	assert.EqualValues(t, NewHistogramBucketSlice(), dest)

	// Test CopyTo larger slice
	generateTestHistogramBucketSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestHistogramBucketSlice(), dest)

	// Test CopyTo same size slice
	generateTestHistogramBucketSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestHistogramBucketSlice(), dest)
}

func TestHistogramBucketSlice_Resize(t *testing.T) {
	es := generateTestHistogramBucketSlice()
	emptyVal := NewHistogramBucket()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.HistogramDataPoint_Bucket]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewHistogramBucketSlice(), es)
}

func TestHistogramBucketSlice_Append(t *testing.T) {
	es := generateTestHistogramBucketSlice()
	emptyVal := NewHistogramBucket()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewHistogramBucket()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestHistogramBucket_InitEmpty(t *testing.T) {
	ms := NewHistogramBucket()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestHistogramBucket_CopyTo(t *testing.T) {
	ms := NewHistogramBucket()
	NewHistogramBucket().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestHistogramBucket().CopyTo(ms)
	assert.EqualValues(t, generateTestHistogramBucket(), ms)
}

func TestHistogramBucket_Count(t *testing.T) {
	ms := NewHistogramBucket()
	ms.InitEmpty()
	assert.EqualValues(t, uint64(0), ms.Count())
	testValCount := uint64(17)
	ms.SetCount(testValCount)
	assert.EqualValues(t, testValCount, ms.Count())
}

func TestHistogramBucket_Exemplar(t *testing.T) {
	ms := NewHistogramBucket()
	ms.InitEmpty()
	assert.EqualValues(t, true, ms.Exemplar().IsNil())
	ms.Exemplar().InitEmpty()
	assert.EqualValues(t, false, ms.Exemplar().IsNil())
	fillTestHistogramBucketExemplar(ms.Exemplar())
	assert.EqualValues(t, generateTestHistogramBucketExemplar(), ms.Exemplar())
}

func TestHistogramBucketExemplar_InitEmpty(t *testing.T) {
	ms := NewHistogramBucketExemplar()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestHistogramBucketExemplar_CopyTo(t *testing.T) {
	ms := NewHistogramBucketExemplar()
	NewHistogramBucketExemplar().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestHistogramBucketExemplar().CopyTo(ms)
	assert.EqualValues(t, generateTestHistogramBucketExemplar(), ms)
}

func TestHistogramBucketExemplar_Timestamp(t *testing.T) {
	ms := NewHistogramBucketExemplar()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := pdata.TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())
}

func TestHistogramBucketExemplar_Value(t *testing.T) {
	ms := NewHistogramBucketExemplar()
	ms.InitEmpty()
	assert.EqualValues(t, float64(0.0), ms.Value())
	testValValue := float64(17.13)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())
}

func TestHistogramBucketExemplar_Attachments(t *testing.T) {
	ms := NewHistogramBucketExemplar()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.NewStringMap(), ms.Attachments())
	fillTestStringMap(ms.Attachments())
	testValAttachments := generateTestStringMap()
	assert.EqualValues(t, testValAttachments, ms.Attachments())
}

func TestSummaryDataPointSlice(t *testing.T) {
	es := NewSummaryDataPointSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newSummaryDataPointSlice(&[]*otlpmetrics.SummaryDataPoint{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewSummaryDataPoint()
	emptyVal.InitEmpty()
	testVal := generateTestSummaryDataPoint()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestSummaryDataPoint(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestSummaryDataPointSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestSummaryDataPointSlice()
	dest := NewSummaryDataPointSlice()
	src := generateTestSummaryDataPointSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestSummaryDataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestSummaryDataPointSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestSummaryDataPointSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSummaryDataPointSlice_CopyTo(t *testing.T) {
	dest := NewSummaryDataPointSlice()
	// Test CopyTo to empty
	NewSummaryDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, NewSummaryDataPointSlice(), dest)

	// Test CopyTo larger slice
	generateTestSummaryDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestSummaryDataPointSlice(), dest)

	// Test CopyTo same size slice
	generateTestSummaryDataPointSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestSummaryDataPointSlice(), dest)
}

func TestSummaryDataPointSlice_Resize(t *testing.T) {
	es := generateTestSummaryDataPointSlice()
	emptyVal := NewSummaryDataPoint()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.SummaryDataPoint]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.SummaryDataPoint]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.SummaryDataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.SummaryDataPoint]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewSummaryDataPointSlice(), es)
}

func TestSummaryDataPointSlice_Append(t *testing.T) {
	es := generateTestSummaryDataPointSlice()
	emptyVal := NewSummaryDataPoint()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewSummaryDataPoint()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestSummaryDataPoint_InitEmpty(t *testing.T) {
	ms := NewSummaryDataPoint()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestSummaryDataPoint_CopyTo(t *testing.T) {
	ms := NewSummaryDataPoint()
	NewSummaryDataPoint().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestSummaryDataPoint().CopyTo(ms)
	assert.EqualValues(t, generateTestSummaryDataPoint(), ms)
}

func TestSummaryDataPoint_LabelsMap(t *testing.T) {
	ms := NewSummaryDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.NewStringMap(), ms.LabelsMap())
	fillTestStringMap(ms.LabelsMap())
	testValLabelsMap := generateTestStringMap()
	assert.EqualValues(t, testValLabelsMap, ms.LabelsMap())
}

func TestSummaryDataPoint_StartTime(t *testing.T) {
	ms := NewSummaryDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.StartTime())
	testValStartTime := pdata.TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())
}

func TestSummaryDataPoint_Timestamp(t *testing.T) {
	ms := NewSummaryDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, pdata.TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := pdata.TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())
}

func TestSummaryDataPoint_Count(t *testing.T) {
	ms := NewSummaryDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, uint64(0), ms.Count())
	testValCount := uint64(17)
	ms.SetCount(testValCount)
	assert.EqualValues(t, testValCount, ms.Count())
}

func TestSummaryDataPoint_Sum(t *testing.T) {
	ms := NewSummaryDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, float64(0.0), ms.Sum())
	testValSum := float64(17.13)
	ms.SetSum(testValSum)
	assert.EqualValues(t, testValSum, ms.Sum())
}

func TestSummaryDataPoint_ValueAtPercentiles(t *testing.T) {
	ms := NewSummaryDataPoint()
	ms.InitEmpty()
	assert.EqualValues(t, NewSummaryValueAtPercentileSlice(), ms.ValueAtPercentiles())
	fillTestSummaryValueAtPercentileSlice(ms.ValueAtPercentiles())
	testValValueAtPercentiles := generateTestSummaryValueAtPercentileSlice()
	assert.EqualValues(t, testValValueAtPercentiles, ms.ValueAtPercentiles())
}

func TestSummaryValueAtPercentileSlice(t *testing.T) {
	es := NewSummaryValueAtPercentileSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newSummaryValueAtPercentileSlice(&[]*otlpmetrics.SummaryDataPoint_ValueAtPercentile{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := NewSummaryValueAtPercentile()
	emptyVal.InitEmpty()
	testVal := generateTestSummaryValueAtPercentile()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestSummaryValueAtPercentile(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func TestSummaryValueAtPercentileSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestSummaryValueAtPercentileSlice()
	dest := NewSummaryValueAtPercentileSlice()
	src := generateTestSummaryValueAtPercentileSlice()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestSummaryValueAtPercentileSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestSummaryValueAtPercentileSlice(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestSummaryValueAtPercentileSlice().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSummaryValueAtPercentileSlice_CopyTo(t *testing.T) {
	dest := NewSummaryValueAtPercentileSlice()
	// Test CopyTo to empty
	NewSummaryValueAtPercentileSlice().CopyTo(dest)
	assert.EqualValues(t, NewSummaryValueAtPercentileSlice(), dest)

	// Test CopyTo larger slice
	generateTestSummaryValueAtPercentileSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestSummaryValueAtPercentileSlice(), dest)

	// Test CopyTo same size slice
	generateTestSummaryValueAtPercentileSlice().CopyTo(dest)
	assert.EqualValues(t, generateTestSummaryValueAtPercentileSlice(), dest)
}

func TestSummaryValueAtPercentileSlice_Resize(t *testing.T) {
	es := generateTestSummaryValueAtPercentileSlice()
	emptyVal := NewSummaryValueAtPercentile()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlpmetrics.SummaryDataPoint_ValueAtPercentile]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewSummaryValueAtPercentileSlice(), es)
}

func TestSummaryValueAtPercentileSlice_Append(t *testing.T) {
	es := generateTestSummaryValueAtPercentileSlice()
	emptyVal := NewSummaryValueAtPercentile()
	emptyVal.InitEmpty()

	es.Append(&emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := NewSummaryValueAtPercentile()
	emptyVal2.InitEmpty()

	es.Append(&emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}

func TestSummaryValueAtPercentile_InitEmpty(t *testing.T) {
	ms := NewSummaryValueAtPercentile()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func TestSummaryValueAtPercentile_CopyTo(t *testing.T) {
	ms := NewSummaryValueAtPercentile()
	NewSummaryValueAtPercentile().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTestSummaryValueAtPercentile().CopyTo(ms)
	assert.EqualValues(t, generateTestSummaryValueAtPercentile(), ms)
}

func TestSummaryValueAtPercentile_Percentile(t *testing.T) {
	ms := NewSummaryValueAtPercentile()
	ms.InitEmpty()
	assert.EqualValues(t, float64(0.0), ms.Percentile())
	testValPercentile := float64(0.90)
	ms.SetPercentile(testValPercentile)
	assert.EqualValues(t, testValPercentile, ms.Percentile())
}

func TestSummaryValueAtPercentile_Value(t *testing.T) {
	ms := NewSummaryValueAtPercentile()
	ms.InitEmpty()
	assert.EqualValues(t, float64(0.0), ms.Value())
	testValValue := float64(17.13)
	ms.SetValue(testValValue)
	assert.EqualValues(t, testValValue, ms.Value())
}

func generateTestResourceMetricsSlice() ResourceMetricsSlice {
	tv := NewResourceMetricsSlice()
	fillTestResourceMetricsSlice(tv)
	return tv
}

func fillTestResourceMetricsSlice(tv ResourceMetricsSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestResourceMetrics(tv.At(i))
	}
}

func generateTestResourceMetrics() ResourceMetrics {
	tv := NewResourceMetrics()
	tv.InitEmpty()
	fillTestResourceMetrics(tv)
	return tv
}

func fillTestResourceMetrics(tv ResourceMetrics) {
	tv.Resource().InitEmpty()
	fillTestResource(tv.Resource())
	fillTestInstrumentationLibraryMetricsSlice(tv.InstrumentationLibraryMetrics())
}

func generateTestInstrumentationLibraryMetricsSlice() InstrumentationLibraryMetricsSlice {
	tv := NewInstrumentationLibraryMetricsSlice()
	fillTestInstrumentationLibraryMetricsSlice(tv)
	return tv
}

func fillTestInstrumentationLibraryMetricsSlice(tv InstrumentationLibraryMetricsSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestInstrumentationLibraryMetrics(tv.At(i))
	}
}

func generateTestInstrumentationLibraryMetrics() InstrumentationLibraryMetrics {
	tv := NewInstrumentationLibraryMetrics()
	tv.InitEmpty()
	fillTestInstrumentationLibraryMetrics(tv)
	return tv
}

func fillTestInstrumentationLibraryMetrics(tv InstrumentationLibraryMetrics) {
	tv.InstrumentationLibrary().InitEmpty()
	fillTestInstrumentationLibrary(tv.InstrumentationLibrary())
	fillTestMetricSlice(tv.Metrics())
}

func generateTestMetricSlice() MetricSlice {
	tv := NewMetricSlice()
	fillTestMetricSlice(tv)
	return tv
}

func fillTestMetricSlice(tv MetricSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestMetric(tv.At(i))
	}
}

func generateTestMetric() Metric {
	tv := NewMetric()
	tv.InitEmpty()
	fillTestMetric(tv)
	return tv
}

func fillTestMetric(tv Metric) {
	tv.MetricDescriptor().InitEmpty()
	fillTestMetricDescriptor(tv.MetricDescriptor())
	fillTestInt64DataPointSlice(tv.Int64DataPoints())
	fillTestDoubleDataPointSlice(tv.DoubleDataPoints())
	fillTestHistogramDataPointSlice(tv.HistogramDataPoints())
	fillTestSummaryDataPointSlice(tv.SummaryDataPoints())
}

func generateTestMetricDescriptor() MetricDescriptor {
	tv := NewMetricDescriptor()
	tv.InitEmpty()
	fillTestMetricDescriptor(tv)
	return tv
}

func fillTestMetricDescriptor(tv MetricDescriptor) {
	tv.SetName("test_name")
	tv.SetDescription("test_description")
	tv.SetUnit("1")
	tv.SetType(MetricTypeInt64)
}

func generateTestInt64DataPointSlice() Int64DataPointSlice {
	tv := NewInt64DataPointSlice()
	fillTestInt64DataPointSlice(tv)
	return tv
}

func fillTestInt64DataPointSlice(tv Int64DataPointSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestInt64DataPoint(tv.At(i))
	}
}

func generateTestInt64DataPoint() Int64DataPoint {
	tv := NewInt64DataPoint()
	tv.InitEmpty()
	fillTestInt64DataPoint(tv)
	return tv
}

func fillTestInt64DataPoint(tv Int64DataPoint) {
	fillTestStringMap(tv.LabelsMap())
	tv.SetStartTime(pdata.TimestampUnixNano(1234567890))
	tv.SetTimestamp(pdata.TimestampUnixNano(1234567890))
	tv.SetValue(int64(-17))
}

func generateTestDoubleDataPointSlice() DoubleDataPointSlice {
	tv := NewDoubleDataPointSlice()
	fillTestDoubleDataPointSlice(tv)
	return tv
}

func fillTestDoubleDataPointSlice(tv DoubleDataPointSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestDoubleDataPoint(tv.At(i))
	}
}

func generateTestDoubleDataPoint() DoubleDataPoint {
	tv := NewDoubleDataPoint()
	tv.InitEmpty()
	fillTestDoubleDataPoint(tv)
	return tv
}

func fillTestDoubleDataPoint(tv DoubleDataPoint) {
	fillTestStringMap(tv.LabelsMap())
	tv.SetStartTime(pdata.TimestampUnixNano(1234567890))
	tv.SetTimestamp(pdata.TimestampUnixNano(1234567890))
	tv.SetValue(float64(17.13))
}

func generateTestHistogramDataPointSlice() HistogramDataPointSlice {
	tv := NewHistogramDataPointSlice()
	fillTestHistogramDataPointSlice(tv)
	return tv
}

func fillTestHistogramDataPointSlice(tv HistogramDataPointSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestHistogramDataPoint(tv.At(i))
	}
}

func generateTestHistogramDataPoint() HistogramDataPoint {
	tv := NewHistogramDataPoint()
	tv.InitEmpty()
	fillTestHistogramDataPoint(tv)
	return tv
}

func fillTestHistogramDataPoint(tv HistogramDataPoint) {
	fillTestStringMap(tv.LabelsMap())
	tv.SetStartTime(pdata.TimestampUnixNano(1234567890))
	tv.SetTimestamp(pdata.TimestampUnixNano(1234567890))
	tv.SetCount(uint64(17))
	tv.SetSum(float64(17.13))
	fillTestHistogramBucketSlice(tv.Buckets())
	tv.SetExplicitBounds([]float64{1, 2, 3})
}

func generateTestHistogramBucketSlice() HistogramBucketSlice {
	tv := NewHistogramBucketSlice()
	fillTestHistogramBucketSlice(tv)
	return tv
}

func fillTestHistogramBucketSlice(tv HistogramBucketSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestHistogramBucket(tv.At(i))
	}
}

func generateTestHistogramBucket() HistogramBucket {
	tv := NewHistogramBucket()
	tv.InitEmpty()
	fillTestHistogramBucket(tv)
	return tv
}

func fillTestHistogramBucket(tv HistogramBucket) {
	tv.SetCount(uint64(17))
	tv.Exemplar().InitEmpty()
	fillTestHistogramBucketExemplar(tv.Exemplar())
}

func generateTestHistogramBucketExemplar() HistogramBucketExemplar {
	tv := NewHistogramBucketExemplar()
	tv.InitEmpty()
	fillTestHistogramBucketExemplar(tv)
	return tv
}

func fillTestHistogramBucketExemplar(tv HistogramBucketExemplar) {
	tv.SetTimestamp(pdata.TimestampUnixNano(1234567890))
	tv.SetValue(float64(17.13))
	fillTestStringMap(tv.Attachments())
}

func generateTestSummaryDataPointSlice() SummaryDataPointSlice {
	tv := NewSummaryDataPointSlice()
	fillTestSummaryDataPointSlice(tv)
	return tv
}

func fillTestSummaryDataPointSlice(tv SummaryDataPointSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestSummaryDataPoint(tv.At(i))
	}
}

func generateTestSummaryDataPoint() SummaryDataPoint {
	tv := NewSummaryDataPoint()
	tv.InitEmpty()
	fillTestSummaryDataPoint(tv)
	return tv
}

func fillTestSummaryDataPoint(tv SummaryDataPoint) {
	fillTestStringMap(tv.LabelsMap())
	tv.SetStartTime(pdata.TimestampUnixNano(1234567890))
	tv.SetTimestamp(pdata.TimestampUnixNano(1234567890))
	tv.SetCount(uint64(17))
	tv.SetSum(float64(17.13))
	fillTestSummaryValueAtPercentileSlice(tv.ValueAtPercentiles())
}

func generateTestSummaryValueAtPercentileSlice() SummaryValueAtPercentileSlice {
	tv := NewSummaryValueAtPercentileSlice()
	fillTestSummaryValueAtPercentileSlice(tv)
	return tv
}

func fillTestSummaryValueAtPercentileSlice(tv SummaryValueAtPercentileSlice) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTestSummaryValueAtPercentile(tv.At(i))
	}
}

func generateTestSummaryValueAtPercentile() SummaryValueAtPercentile {
	tv := NewSummaryValueAtPercentile()
	tv.InitEmpty()
	fillTestSummaryValueAtPercentile(tv)
	return tv
}

func fillTestSummaryValueAtPercentile(tv SummaryValueAtPercentile) {
	tv.SetPercentile(float64(0.90))
	tv.SetValue(float64(17.13))
}

func generateTestInstrumentationLibrary() pdata.InstrumentationLibrary {
	tv := pdata.NewInstrumentationLibrary()
	tv.InitEmpty()
	fillTestInstrumentationLibrary(tv)
	return tv
}

func fillTestInstrumentationLibrary(tv pdata.InstrumentationLibrary) {
	tv.SetName("test_name")
	tv.SetVersion("test_version")
}

func generateTestStringMap() pdata.StringMap {
	sm := pdata.NewStringMap()
	fillTestStringMap(sm)
	return sm
}

func fillTestStringMap(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{
		"k": "v",
	})
}

func generateTestResource() pdata.Resource {
	tv := pdata.NewResource()
	tv.InitEmpty()
	fillTestResource(tv)
	return tv
}

func fillTestResource(tv pdata.Resource) {
	fillTestAttributeMap(tv.Attributes())
}

func fillTestAttributeMap(dest pdata.AttributeMap) {
	dest.InitFromMap(map[string]pdata.AttributeValue{
		"k": pdata.NewAttributeValueString("v"),
	})
}
