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

package batchprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestSplitMetrics_noop(t *testing.T) {
	td := testdata.GenerateMetricsManyMetricsSameResource(20)
	splitSize := 40
	split := splitMetrics(splitSize, td)
	assert.Equal(t, td, split)

	i := 0
	td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().RemoveIf(func(_ pdata.Metric) bool {
		i++
		return i > 5
	})
	assert.EqualValues(t, td, split)
}

func TestSplitMetrics(t *testing.T) {
	md := testdata.GenerateMetricsManyMetricsSameResource(20)
	metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	dataPointCount := metricDataPointCount(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDataPointCount(metrics.At(i)))
	}
	cp := pdata.NewMetrics()
	cpMetrics := cp.ResourceMetrics().AppendEmpty().InstrumentationLibraryMetrics().AppendEmpty().Metrics()
	cpMetrics.EnsureCapacity(5)
	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).InstrumentationLibrary().CopyTo(
		cp.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).InstrumentationLibrary())
	md.ResourceMetrics().At(0).Resource().CopyTo(
		cp.ResourceMetrics().At(0).Resource())
	metrics.At(0).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(1).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(2).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(3).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(4).CopyTo(cpMetrics.AppendEmpty())

	splitMetricCount := 5
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, md)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, cp, split)
	assert.Equal(t, 15, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 10, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-5", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-9", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 5, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-10", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-14", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 5, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-15", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-19", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsMultipleResourceSpans(t *testing.T) {
	md := testdata.GenerateMetricsManyMetricsSameResource(20)
	metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	dataPointCount := metricDataPointCount(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDataPointCount(metrics.At(i)))
	}
	// add second index to resource metrics
	testdata.GenerateMetricsManyMetricsSameResource(20).
		ResourceMetrics().At(0).CopyTo(md.ResourceMetrics().AppendEmpty())
	metrics = md.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(1, i))
	}

	splitMetricCount := 5
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, md)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, 35, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsMultipleResourceSpans_SplitSizeGreaterThanMetricSize(t *testing.T) {
	td := testdata.GenerateMetricsManyMetricsSameResource(20)
	metrics := td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	dataPointCount := metricDataPointCount(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDataPointCount(metrics.At(i)))
	}
	// add second index to resource metrics
	testdata.GenerateMetricsManyMetricsSameResource(20).
		ResourceMetrics().At(0).CopyTo(td.ResourceMetrics().AppendEmpty())
	metrics = td.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(1, i))
	}

	splitMetricCount := 25
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, td)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, 40-splitMetricCount, td.MetricCount())
	assert.Equal(t, 1, td.ResourceMetrics().Len())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-19", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(19).Name())
	assert.Equal(t, "test-metric-int-1-0", split.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-1-4", split.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())
}

func BenchmarkSplitMetrics(b *testing.B) {
	md := pdata.NewMetrics()
	rms := md.ResourceMetrics()
	for i := 0; i < 20; i++ {
		testdata.GenerateMetricsManyMetricsSameResource(20).ResourceMetrics().MoveAndAppendTo(md.ResourceMetrics())
		ms := rms.At(rms.Len() - 1).InstrumentationLibraryMetrics().At(0).Metrics()
		for i := 0; i < ms.Len(); i++ {
			ms.At(i).SetName(getTestMetricName(1, i))
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cloneReq := md.Clone()
		split := splitMetrics(128, cloneReq)
		if split.MetricCount() != 128 || cloneReq.MetricCount() != 400-128 {
			b.Fail()
		}
	}
}

func BenchmarkCloneMetrics(b *testing.B) {
	md := pdata.NewMetrics()
	rms := md.ResourceMetrics()
	for i := 0; i < 20; i++ {
		testdata.GenerateMetricsManyMetricsSameResource(20).ResourceMetrics().MoveAndAppendTo(md.ResourceMetrics())
		ms := rms.At(rms.Len() - 1).InstrumentationLibraryMetrics().At(0).Metrics()
		for i := 0; i < ms.Len(); i++ {
			ms.At(i).SetName(getTestMetricName(1, i))
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cloneReq := md.Clone()
		if cloneReq.MetricCount() != 400 {
			b.Fail()
		}
	}
}

func TestSplitMetricsUneven(t *testing.T) {
	md := testdata.GenerateMetricsManyMetricsSameResource(10)
	metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	dataPointCount := 2
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDataPointCount(metrics.At(i)))
	}

	splitSize := 9
	split := splitMetrics(splitSize, md)
	assert.Equal(t, 5, split.MetricCount())
	assert.Equal(t, 6, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 5, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-8", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, "test-metric-int-0-9", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
}

func TestSplitMetricsBatchSizeSmallerThanDataPointCount(t *testing.T) {
	md := testdata.GenerateMetricsManyMetricsSameResource(2)
	metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	dataPointCount := 2
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDataPointCount(metrics.At(i)))
	}

	splitSize := 1
	split := splitMetrics(splitSize, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 2, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-1", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-1", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
}
