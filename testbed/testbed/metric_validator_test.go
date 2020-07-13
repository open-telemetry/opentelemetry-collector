// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testbed

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
)

func TestSameMetrics(t *testing.T) {
	expected := data.NewMetricData()
	populateResourceMetrics(expected.ResourceMetrics(), defaultCfg())
	actual := data.NewMetricData()
	populateResourceMetrics(actual.ResourceMetrics(), defaultCfg())
	diffs := diffMetricData(expected, actual)
	assert.Nil(t, diffs)
}

func TestDifferentValues(t *testing.T) {
	expected := data.NewMetricData()
	populateResourceMetrics(expected.ResourceMetrics(), defaultCfg())
	actual := data.NewMetricData()
	cfg := defaultCfg()
	cfg.ptVal = 2
	populateResourceMetrics(actual.ResourceMetrics(), cfg)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 1, len(diffs))
}

func TestDifferentNumPts(t *testing.T) {
	expected := data.NewMetricData()
	populateResourceMetrics(expected.ResourceMetrics(), defaultCfg())
	actual := data.NewMetricData()
	cfg := defaultCfg()
	cfg.numPts = 2
	populateResourceMetrics(actual.ResourceMetrics(), cfg)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 1, len(diffs))
}

func TestDifferentPtTypes(t *testing.T) {
	expected := data.NewMetricData()
	populateResourceMetrics(expected.ResourceMetrics(), defaultCfg())
	actual := data.NewMetricData()
	cfg := defaultCfg()
	cfg.ptsType = "double"
	populateResourceMetrics(actual.ResourceMetrics(), cfg)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 2, len(diffs))
}

func TestHistogram(t *testing.T) {
	expected := data.NewMetricData()
	cfg1 := defaultCfg()
	cfg1.ptsType = "histogram"
	populateResourceMetrics(expected.ResourceMetrics(), cfg1)
	actual := data.NewMetricData()
	cfg2 := defaultCfg()
	cfg2.ptsType = "histogram"
	cfg2.ptVal = 2
	populateResourceMetrics(actual.ResourceMetrics(), cfg2)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 1, len(diffs))
}

func TestSummary(t *testing.T) {
	expected := data.NewMetricData()
	cfg1 := defaultCfg()
	cfg1.ptsType = "summary"
	populateResourceMetrics(expected.ResourceMetrics(), cfg1)
	actual := data.NewMetricData()
	cfg2 := defaultCfg()
	cfg2.ptsType = "summary"
	cfg2.ptVal = 2
	populateResourceMetrics(actual.ResourceMetrics(), cfg2)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 1, len(diffs))
}

// populators

type metricCfg struct {
	numResourceMetrics int
	resourceAttrName   string
	resourceAttrVal    string
	numIlm             int
	numMetrics         int
	name               string
	desc               string
	unit               string
	numPts             int
	ptVal              int64
	ptsType            string
	numBuckets         int
}

func defaultCfg() metricCfg {
	cfg := metricCfg{
		numResourceMetrics: 1,
		resourceAttrName:   "attr-name",
		resourceAttrVal:    "attr-val",
		numIlm:             1,
		numMetrics:         1,
		name:               "my-name",
		desc:               "my-desc",
		unit:               "my-units",
		numPts:             1,
		ptVal:              1,
		ptsType:            "int",
		numBuckets:         1,
	}
	return cfg
}

func populateResourceMetrics(rmSlice pdata.ResourceMetricsSlice, cfg metricCfg) {
	rmSlice.Resize(cfg.numResourceMetrics)
	for i := 0; i < cfg.numResourceMetrics; i++ {
		rm := rmSlice.At(i)
		resource := rm.Resource()
		resource.InitEmpty()
		resource.Attributes().Insert(cfg.resourceAttrName, pdata.NewAttributeValueString(cfg.resourceAttrVal))
		populateIlm(rm, cfg)
	}
}

func populateIlm(rm pdata.ResourceMetrics, cfg metricCfg) {
	ilmSlice := rm.InstrumentationLibraryMetrics()
	ilmSlice.Resize(cfg.numIlm)
	for i := 0; i < cfg.numIlm; i++ {
		ilm := ilmSlice.At(i)
		populateMetrics(ilm, cfg)
	}
}

func populateMetrics(ilm pdata.InstrumentationLibraryMetrics, cfg metricCfg) {
	metricSlice := ilm.Metrics()
	for i := 0; i < cfg.numMetrics; i++ {
		metricSlice.Resize(cfg.numMetrics)
		metric := metricSlice.At(i)
		metric.InitEmpty()
		populateMetricDesc(metric, cfg)
		switch cfg.ptsType {
		case "int":
			populateIntPoints(metric, cfg)
		case "double":
			populateDblPoints(metric, cfg)
		case "histogram":
			populateHistogramPoints(metric, cfg)
		case "summary":
			populateSummaryPoints(metric, cfg)
		}
	}
}

func populateMetricDesc(metric pdata.Metric, cfg metricCfg) {
	mDesc := metric.MetricDescriptor()
	mDesc.InitEmpty()
	mDesc.SetName(cfg.name)
	mDesc.SetDescription(cfg.desc)
	mDesc.SetUnit(cfg.unit)
	mDesc.SetType(pdata.MetricTypeMonotonicInt64)
}

func populateHistogramPoints(metric pdata.Metric, cfg metricCfg) {
	pts := metric.HistogramDataPoints()
	pts.Resize(cfg.numPts)
	for i := 0; i < cfg.numPts; i++ {
		pt := pts.At(i)
		pt.SetCount(1) // fixme?
		pt.SetSum(1)
		pt.SetExplicitBounds([]float64{1})
		buckets := pt.Buckets()
		buckets.Resize(cfg.numBuckets)
		for j := 0; j < cfg.numBuckets; j++ {
			bucket := buckets.At(j)
			bucket.SetCount(uint64(j))
			ex := bucket.Exemplar()
			ex.InitEmpty()
			ex.SetValue(float64(cfg.ptVal))
		}
	}
}

func populateSummaryPoints(metric pdata.Metric, cfg metricCfg) {
	pts := metric.SummaryDataPoints()
	pts.Resize(cfg.numPts)
	for i := 0; i < cfg.numPts; i++ {
		pt := pts.At(i)
		pt.SetCount(uint64(i))
		pt.SetSum(float64(cfg.ptVal))
	}
}

func populateIntPoints(metric pdata.Metric, cfg metricCfg) {
	pts := metric.Int64DataPoints()
	pts.Resize(cfg.numPts)
	for i := 0; i < cfg.numPts; i++ {
		pts.At(i).SetValue(cfg.ptVal + int64(i))
	}
}

func populateDblPoints(metric pdata.Metric, cfg metricCfg) {
	pts := metric.DoubleDataPoints()
	pts.Resize(cfg.numPts)
	for i := 0; i < cfg.numPts; i++ {
		v := cfg.ptVal + int64(i)
		pts.At(i).SetValue(float64(v))
	}
}
