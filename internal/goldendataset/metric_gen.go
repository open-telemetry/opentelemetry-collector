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

package goldendataset

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
)

// Simple utilities for generating metrics for testing

// MetricCfg holds parameters for generating dummy metrics for testing. Set values on this struct to generate
// metrics with the corresponding number/type of attributes and pass into MetricDataFromCfg to generate metrics.
type MetricCfg struct {
	// The type of metric to generate
	MetricDescriptorType dataold.MetricType
	// A prefix for every metric name
	MetricNamePrefix string
	// The number of instrumentation library metrics per resource
	NumILMPerResource int
	// The size of the MetricSlice and number of Metrics
	NumMetricsPerILM int
	// The number of labels on the LabelsMap associated with each point
	NumPtLabels int
	// The number of points to generate per Metric
	NumPtsPerMetric int
	// The number of Attributes to insert into each Resource's AttributesMap
	NumResourceAttrs int
	// The number of ResourceMetrics for the single MetricData generated
	NumResourceMetrics int
	// The base value for each point
	PtVal int
	// The start time for each point
	StartTime uint64
	// The duration of the steps between each generated point starting at StartTime
	StepSize uint64
}

// DefaultCfg produces a MetricCfg with default values. These should be good enough to produce sane
// (but boring) metrics, and can be used as a starting point for making alterations.
func DefaultCfg() MetricCfg {
	return MetricCfg{
		MetricDescriptorType: dataold.MetricTypeInt64,
		MetricNamePrefix:     "",
		NumILMPerResource:    1,
		NumMetricsPerILM:     1,
		NumPtLabels:          1,
		NumPtsPerMetric:      1,
		NumResourceAttrs:     1,
		NumResourceMetrics:   1,
		PtVal:                1,
		StartTime:            940000000000000000,
		StepSize:             42,
	}
}

// DefaultMetricData produces MetricData with a default config.
func DefaultMetricData() dataold.MetricData {
	return MetricDataFromCfg(DefaultCfg())
}

// MetricDataFromCfg produces MetricData with the passed-in config.
func MetricDataFromCfg(cfg MetricCfg) dataold.MetricData {
	return newMetricGenerator().genMetricDataFromCfg(cfg)
}

type metricGenerator struct {
	metricID int
}

func newMetricGenerator() *metricGenerator {
	return &metricGenerator{}
}

func (g *metricGenerator) genMetricDataFromCfg(cfg MetricCfg) dataold.MetricData {
	md := dataold.NewMetricData()
	rms := md.ResourceMetrics()
	rms.Resize(cfg.NumResourceMetrics)
	for i := 0; i < cfg.NumResourceMetrics; i++ {
		rm := rms.At(i)
		resource := rm.Resource()
		resource.InitEmpty()
		for j := 0; j < cfg.NumResourceAttrs; j++ {
			resource.Attributes().Insert(
				fmt.Sprintf("resource-attr-name-%d", j),
				pdata.NewAttributeValueString(fmt.Sprintf("resource-attr-val-%d", j)),
			)
		}
		g.populateIlm(cfg, rm)
	}
	return md
}

func (g *metricGenerator) populateIlm(cfg MetricCfg, rm dataold.ResourceMetrics) {
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(cfg.NumILMPerResource)
	for i := 0; i < cfg.NumILMPerResource; i++ {
		ilm := ilms.At(i)
		g.populateMetrics(cfg, ilm)
	}
}

func (g *metricGenerator) populateMetrics(cfg MetricCfg, ilm dataold.InstrumentationLibraryMetrics) {
	metrics := ilm.Metrics()
	metrics.Resize(cfg.NumMetricsPerILM)
	for i := 0; i < cfg.NumMetricsPerILM; i++ {
		metric := metrics.At(i)
		metric.InitEmpty()
		g.populateMetricDesc(cfg, metric)
		switch cfg.MetricDescriptorType {
		case dataold.MetricTypeInt64, dataold.MetricTypeMonotonicInt64:
			populateIntPoints(cfg, metric)
		case dataold.MetricTypeDouble, dataold.MetricTypeMonotonicDouble:
			populateDblPoints(cfg, metric)
		case dataold.MetricTypeHistogram:
			populateHistogramPoints(cfg, metric)
		case dataold.MetricTypeSummary:
			populateSummaryPoints(cfg, metric)
		}
	}
}

func (g *metricGenerator) populateMetricDesc(cfg MetricCfg, metric dataold.Metric) {
	desc := metric.MetricDescriptor()
	desc.InitEmpty()
	desc.SetName(fmt.Sprintf("%smetric_%d", cfg.MetricNamePrefix, g.metricID))
	g.metricID++
	desc.SetDescription("my-md-description")
	desc.SetUnit("my-md-units")
	desc.SetType(cfg.MetricDescriptorType)
}

func populateIntPoints(cfg MetricCfg, metric dataold.Metric) {
	pts := metric.Int64DataPoints()
	pts.Resize(cfg.NumPtsPerMetric)
	for i := 0; i < cfg.NumPtsPerMetric; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.StartTime))
		pt.SetTimestamp(getTimestamp(cfg.StartTime, cfg.StepSize, i))
		pt.SetValue(int64(cfg.PtVal + i))
		populatePtLabels(cfg, pt.LabelsMap())
	}
}

func populateDblPoints(cfg MetricCfg, metric dataold.Metric) {
	pts := metric.DoubleDataPoints()
	pts.Resize(cfg.NumPtsPerMetric)
	for i := 0; i < cfg.NumPtsPerMetric; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.StartTime))
		pt.SetTimestamp(getTimestamp(cfg.StartTime, cfg.StepSize, i))
		pt.SetValue(float64(cfg.PtVal + i))
		populatePtLabels(cfg, pt.LabelsMap())
	}
}

func populateHistogramPoints(cfg MetricCfg, metric dataold.Metric) {
	pts := metric.HistogramDataPoints()
	pts.Resize(cfg.NumPtsPerMetric)
	for i := 0; i < cfg.NumPtsPerMetric; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.StartTime))
		ts := getTimestamp(cfg.StartTime, cfg.StepSize, i)
		pt.SetTimestamp(ts)
		populatePtLabels(cfg, pt.LabelsMap())
		setHistogramBounds(pt, 1, 2, 3, 4, 5)
		addHistogramVal(pt, 1, ts)
		for i := 0; i < cfg.PtVal; i++ {
			addHistogramVal(pt, 3, ts)
		}
		addHistogramVal(pt, 5, ts)
	}
}

func setHistogramBounds(hdp dataold.HistogramDataPoint, bounds ...float64) {
	hdp.Buckets().Resize(len(bounds))
	hdp.SetExplicitBounds(bounds)
}

func addHistogramVal(hdp dataold.HistogramDataPoint, val float64, ts pdata.TimestampUnixNano) {
	hdp.SetCount(hdp.Count() + 1)
	hdp.SetSum(hdp.Sum() + val)
	buckets := hdp.Buckets()
	bounds := hdp.ExplicitBounds()
	for i := 0; i < len(bounds); i++ {
		bound := bounds[i]
		if val <= bound {
			bucket := buckets.At(i)
			bucket.SetCount(bucket.Count() + 1)
			ex := bucket.Exemplar()
			ex.InitEmpty()
			ex.SetValue(val)
			ex.SetTimestamp(ts)
			break
		}
	}
}

func populateSummaryPoints(cfg MetricCfg, metric dataold.Metric) {
	pts := metric.SummaryDataPoints()
	pts.Resize(cfg.NumPtsPerMetric)
	for i := 0; i < cfg.NumPtsPerMetric; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.StartTime))
		pt.SetTimestamp(getTimestamp(cfg.StartTime, cfg.StepSize, i))
		setSummaryPercentiles(pt, 0, 50, 95)
		addSummaryValue(pt, 55, 0)
		for i := 0; i < cfg.PtVal; i++ {
			addSummaryValue(pt, 70, 1)
		}
		addSummaryValue(pt, 90, 2)
		populatePtLabels(cfg, pt.LabelsMap())
	}
}

func setSummaryPercentiles(pt dataold.SummaryDataPoint, pctiles ...float64) {
	vap := pt.ValueAtPercentiles()
	l := len(pctiles)
	vap.Resize(l)
	for i := 0; i < l; i++ {
		vap.At(i).SetPercentile(pctiles[i])
	}
}

func addSummaryValue(pt dataold.SummaryDataPoint, value float64, pctileIndex int) {
	pt.SetCount(pt.Count() + 1)
	pt.SetSum(pt.Sum() + value)
	vap := pt.ValueAtPercentiles().At(pctileIndex)
	vap.SetValue(vap.Value() + 1)
}

func populatePtLabels(cfg MetricCfg, lm pdata.StringMap) {
	for i := 0; i < cfg.NumPtLabels; i++ {
		k := fmt.Sprintf("pt-label-key-%d", i)
		v := fmt.Sprintf("pt-label-val-%d", i)
		lm.Insert(k, v)
	}
}

func getTimestamp(startTime uint64, stepSize uint64, i int) pdata.TimestampUnixNano {
	return pdata.TimestampUnixNano(startTime + (stepSize * uint64(i+1)))
}
