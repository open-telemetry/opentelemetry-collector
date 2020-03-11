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
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricCount(t *testing.T) {
	md := NewMetricData()
	assert.EqualValues(t, 0, md.MetricCount())

	md.SetResourceMetrics([]ResourceMetrics{NewResourceMetrics()})
	assert.EqualValues(t, 0, md.MetricCount())

	rm := NewResourceMetrics()
	rm.SetMetrics(NewMetricSlice(1))
	md.SetResourceMetrics([]ResourceMetrics{rm})
	assert.EqualValues(t, 1, md.MetricCount())

	rm1 := NewResourceMetrics()
	rm1.SetMetrics(NewMetricSlice(1))
	rm2 := NewResourceMetrics()
	rm3 := NewResourceMetrics()
	rm3.SetMetrics(NewMetricSlice(5))
	md.SetResourceMetrics([]ResourceMetrics{rm1, rm2, rm3})
	assert.EqualValues(t, 6, md.MetricCount())
}

func TestNewMetricSlice(t *testing.T) {
	ms := NewMetricSlice(0)
	assert.EqualValues(t, 0, len(ms))

	n := rand.Intn(10)
	ms = NewMetricSlice(n)
	assert.EqualValues(t, n, len(ms))
	for Metric := range ms {
		assert.NotNil(t, Metric)
	}
}

func TestNewInt64DataPointSlice(t *testing.T) {
	dps := NewInt64DataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewInt64DataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for event := range dps {
		assert.NotNil(t, event)
	}
}

func TestNewDoubleDataPointSlice(t *testing.T) {
	dps := NewDoubleDataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewDoubleDataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for event := range dps {
		assert.NotNil(t, event)
	}
}

func TestNewHistogramDataPointSlice(t *testing.T) {
	dps := NewHistogramDataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewHistogramDataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for event := range dps {
		assert.NotNil(t, event)
	}
}

func TestNewHistogramBucketSlice(t *testing.T) {
	hbs := NewHistogramBucketSlice(0)
	assert.EqualValues(t, 0, len(hbs))

	n := rand.Intn(10)
	hbs = NewHistogramBucketSlice(n)
	assert.EqualValues(t, n, len(hbs))
	for event := range hbs {
		assert.NotNil(t, event)
	}
}

func TestNewSummaryDataPointSlice(t *testing.T) {
	dps := NewSummaryDataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewSummaryDataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for link := range dps {
		assert.NotNil(t, link)
	}
}

func TestNewSummaryValueAtPercentileSlice(t *testing.T) {
	vps := NewSummaryValueAtPercentileSlice(0)
	assert.EqualValues(t, 0, len(vps))

	n := rand.Intn(10)
	vps = NewSummaryValueAtPercentileSlice(n)
	assert.EqualValues(t, n, len(vps))
	for link := range vps {
		assert.NotNil(t, link)
	}
}
