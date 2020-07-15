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

package goldendataset

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
)

// GenerateMetricDatas takes the filename of a PICT-generated file, walks through all of the rows in the PICT
// file and for each row, generates a MetricData object, collecting them and returning them to the caller.
func GenerateMetricDatas(metricPairsFile string) ([]data.MetricData, error) {
	pictData, err := loadPictOutputFile(metricPairsFile)
	if err != nil {
		return nil, err
	}
	var out []data.MetricData
	for i, values := range pictData {
		if i == 0 {
			continue
		}
		metricInputs := PICTMetricInputs{
			NumPtsPerMetric: PICTNumPtsPerMetric(values[0]),
			MetricType:      PICTMetricType(values[1]),
			NumLabels:       PICTNumPtLabels(values[2]),
		}
		md := GenerateMetricData(metricInputs)
		out = append(out, md)
	}
	return out, nil
}

func GenerateMetricData(inputs PICTMetricInputs) data.MetricData {
	cfg := DefaultCfg()
	switch inputs.NumAttrs {
	case AttrsNone:
		cfg.numResourceAttrs = 0
	case AttrsOne:
		cfg.numResourceAttrs = 1
	case AttrsTwo:
		cfg.numResourceAttrs = 2
	}

	switch inputs.NumPtsPerMetric {
	case NumPtsPerMetricOne:
		cfg.NumPts = 1
	case NumPtsPerMetricMany:
		cfg.NumPts = 1024
	}

	switch inputs.MetricType {
	case MetricTypeInt:
		cfg.MetricType = pdata.MetricTypeInt64
	case MetricTypeMonotonicInt:
		cfg.MetricType = pdata.MetricTypeMonotonicInt64
	case MetricTypeDouble:
		cfg.MetricType = pdata.MetricTypeDouble
	case MetricTypeMonotonicDouble:
		cfg.MetricType = pdata.MetricTypeMonotonicDouble
	case MetricTypeHistogram:
		cfg.MetricType = pdata.MetricTypeHistogram
	case MetricTypeSummary:
		cfg.MetricType = pdata.MetricTypeSummary
	}

	switch inputs.NumLabels {
	case LabelsNone:
		cfg.numPtLabels = 0
	case LabelsOne:
		cfg.numPtLabels = 1
	case LabelsMany:
		cfg.numPtLabels = 16
	}

	return GenerateMetricsFromCfg(cfg)
}

type MetricCfg struct {
	NumPts             int
	PtVal              int64
	MetricType         pdata.MetricType
	numResourceMetrics int
	numResourceAttrs   int
	numIlm             int
	numMetrics         int
	name               string
	desc               string
	unit               string
	numBuckets         int
	numPtLabels        int
	startTime          uint64
	stepSize           uint64
}

func DefaultCfg() MetricCfg {
	return MetricCfg{
		numResourceMetrics: 1,
		numResourceAttrs:   1,
		numIlm:             1,
		numMetrics:         1,
		name:               "my-name",
		desc:               "my-desc",
		unit:               "my-units",
		NumPts:             1,
		PtVal:              1,
		MetricType:         pdata.MetricTypeInt64,
		numBuckets:         1,
		numPtLabels:        1,
		startTime:          1000000,
		stepSize:           100000,
	}
}

func GenerateDefaultData() data.MetricData {
	return GenerateMetricsFromCfg(DefaultCfg())
}

func GenerateMetricsFromCfg(cfg MetricCfg) data.MetricData {
	md := data.NewMetricData()
	rmSlice := md.ResourceMetrics()
	rmSlice.Resize(cfg.numResourceMetrics)
	for i := 0; i < cfg.numResourceMetrics; i++ {
		rm := rmSlice.At(i)
		resource := rm.Resource()
		resource.InitEmpty()
		for j := 0; j < cfg.numResourceAttrs; j++ {
			resource.Attributes().Insert(
				fmt.Sprintf("name-%d", j),
				pdata.NewAttributeValueString(fmt.Sprintf("val-%d", j)),
			)
		}
		populateIlm(rm, cfg)
	}
	return md
}

func populateIlm(rm pdata.ResourceMetrics, cfg MetricCfg) {
	ilmSlice := rm.InstrumentationLibraryMetrics()
	ilmSlice.Resize(cfg.numIlm)
	for i := 0; i < cfg.numIlm; i++ {
		ilm := ilmSlice.At(i)
		populateMetrics(ilm, cfg)
	}
}

func populateMetrics(ilm pdata.InstrumentationLibraryMetrics, cfg MetricCfg) {
	metricSlice := ilm.Metrics()
	for i := 0; i < cfg.numMetrics; i++ {
		metricSlice.Resize(cfg.numMetrics)
		metric := metricSlice.At(i)
		metric.InitEmpty()
		populateMetricDesc(metric, cfg)
		switch cfg.MetricType {
		case pdata.MetricTypeInt64, pdata.MetricTypeMonotonicInt64:
			populateIntPoints(metric, cfg)
		case pdata.MetricTypeDouble, pdata.MetricTypeMonotonicDouble:
			populateDblPoints(metric, cfg)
		case pdata.MetricTypeHistogram:
			populateHistogramPoints(metric, cfg)
		case pdata.MetricTypeSummary:
			populateSummaryPoints(metric, cfg)
		}
	}
}

func populateMetricDesc(metric pdata.Metric, cfg MetricCfg) {
	mDesc := metric.MetricDescriptor()
	mDesc.InitEmpty()
	mDesc.SetName(cfg.name)
	mDesc.SetDescription(cfg.desc)
	mDesc.SetUnit(cfg.unit)
	mDesc.SetType(cfg.MetricType)
}

func populateIntPoints(metric pdata.Metric, cfg MetricCfg) {
	pts := metric.Int64DataPoints()
	pts.Resize(cfg.NumPts)
	for i := 0; i < cfg.NumPts; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.startTime))
		pt.SetTimestamp(getTimestamp(cfg.startTime, cfg.stepSize, i))
		pt.SetValue(cfg.PtVal + int64(i))
		populatePtLabels(pt.LabelsMap(), cfg)
	}
}

func populateDblPoints(metric pdata.Metric, cfg MetricCfg) {
	pts := metric.DoubleDataPoints()
	pts.Resize(cfg.NumPts)
	for i := 0; i < cfg.NumPts; i++ {
		v := cfg.PtVal + int64(i)
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.startTime))
		pt.SetTimestamp(getTimestamp(cfg.startTime, cfg.stepSize, i))
		pt.SetValue(float64(v))
		populatePtLabels(pt.LabelsMap(), cfg)
	}
}

func populateHistogramPoints(metric pdata.Metric, cfg MetricCfg) {
	pts := metric.HistogramDataPoints()
	pts.Resize(cfg.NumPts)
	for i := 0; i < cfg.NumPts; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.startTime))
		pt.SetTimestamp(getTimestamp(cfg.startTime, cfg.stepSize, i))
		pt.SetCount(uint64(i)) // fixme?
		pt.SetSum(float64(i))
		pt.SetExplicitBounds([]float64{float64(i)})
		populatePtLabels(pt.LabelsMap(), cfg)
		buckets := pt.Buckets()
		buckets.Resize(cfg.numBuckets)
		for j := 0; j < cfg.numBuckets; j++ {
			bucket := buckets.At(j)
			bucket.SetCount(uint64(j))
			ex := bucket.Exemplar()
			ex.InitEmpty()
			ex.SetValue(float64(cfg.PtVal))
		}
	}
}

func populateSummaryPoints(metric pdata.Metric, cfg MetricCfg) {
	pts := metric.SummaryDataPoints()
	pts.Resize(cfg.NumPts)
	for i := 0; i < cfg.NumPts; i++ {
		pt := pts.At(i)
		pt.SetStartTime(pdata.TimestampUnixNano(cfg.startTime))
		pt.SetTimestamp(getTimestamp(cfg.startTime, cfg.stepSize, i))
		pt.SetCount(uint64(i))
		pt.SetSum(float64(cfg.PtVal))
		populatePtLabels(pt.LabelsMap(), cfg)
	}
}

func populatePtLabels(lm pdata.StringMap, m MetricCfg) {
	for i := 0; i < m.numPtLabels; i++ {
		k := fmt.Sprintf("label-key-%d", i)
		v := fmt.Sprintf("label-value-%d", i)
		lm.Insert(k, v)
	}
}

func getTimestamp(startTime uint64, stepSize uint64, i int) pdata.TimestampUnixNano {
	return pdata.TimestampUnixNano(startTime + (stepSize * uint64(i)))
}
