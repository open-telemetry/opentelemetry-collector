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

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestSplitMetrics_noop(t *testing.T) {
	td := testdata.GenerateMetricsManyMetricsSameResource(20)
	splitSize := 40
	split := splitMetrics(splitSize, td)
	assert.Equal(t, td, split)

	td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(5)
	assert.EqualValues(t, td, split)
}

func TestSplitMetrics(t *testing.T) {
	td := testdata.GenerateMetricsManyMetricsSameResource(20)
	metrics := td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
	}
	cp := pdata.NewMetrics()
	cp.ResourceMetrics().Resize(1)
	cp.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	cp.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(5)
	cpMetrics := cp.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).InstrumentationLibrary().CopyTo(
		cp.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).InstrumentationLibrary())
	td.ResourceMetrics().At(0).Resource().CopyTo(
		cp.ResourceMetrics().At(0).Resource())
	metrics.At(19).CopyTo(cpMetrics.At(0))
	metrics.At(18).CopyTo(cpMetrics.At(1))
	metrics.At(17).CopyTo(cpMetrics.At(2))
	metrics.At(16).CopyTo(cpMetrics.At(3))
	metrics.At(15).CopyTo(cpMetrics.At(4))

	splitSize := 5
	split := splitMetrics(splitSize, td)
	assert.Equal(t, splitSize, split.MetricCount())
	assert.Equal(t, cp, split)
	assert.Equal(t, 15, td.MetricCount())
	assert.Equal(t, "test-metric-int-0-19", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-15", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsMultipleResourceSpans(t *testing.T) {
	td := testdata.GenerateMetricsManyMetricsSameResource(20)
	metrics := td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
	}
	td.ResourceMetrics().Resize(2)
	// add second index to resource metrics
	testdata.GenerateMetricsManyMetricsSameResource(20).
		ResourceMetrics().At(0).CopyTo(td.ResourceMetrics().At(1))
	metrics = td.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(1, i))
	}

	splitSize := 5
	split := splitMetrics(splitSize, td)
	assert.Equal(t, splitSize, split.MetricCount())
	assert.Equal(t, 35, td.MetricCount())
	assert.Equal(t, "test-metric-int-1-19", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-1-15", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsMultipleResourceSpans_split_size_greater_than_metric_size(t *testing.T) {
	td := testdata.GenerateMetricsManyMetricsSameResource(20)
	metrics := td.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
	}
	td.ResourceMetrics().Resize(2)
	// add second index to resource metrics
	testdata.GenerateMetricsManyMetricsSameResource(20).
		ResourceMetrics().At(0).CopyTo(td.ResourceMetrics().At(1))
	metrics = td.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(1, i))
	}

	splitSize := 25
	split := splitMetrics(splitSize, td)
	assert.Equal(t, splitSize, split.MetricCount())
	assert.Equal(t, 40-splitSize, td.MetricCount())
	assert.Equal(t, 1, td.ResourceMetrics().Len())
	assert.Equal(t, "test-metric-int-1-19", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-1-0", split.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(19).Name())
	assert.Equal(t, "test-metric-int-0-19", split.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-15", split.ResourceMetrics().At(1).InstrumentationLibraryMetrics().At(0).Metrics().At(4).Name())
}
