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

	"github.com/prometheus/common/log"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
)

var metricCounter int
var ptCounter int

func GenerateMetrics(metricPairsFile string) ([]data.MetricData, error) {
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
			NumLabels:       PICTNumLabels(values[2]),
		}
		md := GenerateMetric(metricInputs)
		out = append(out, md)
	}
	return out, nil
}

func GenerateMetric(inputs PICTMetricInputs) data.MetricData {
	md := data.NewMetricData()
	rmSlice := md.ResourceMetrics()
	rmSlice.Resize(1)
	rm := rmSlice.At(0)
	rsrc := rm.Resource()
	rsrc.InitEmpty()
	attrs := rsrc.Attributes()

	var numResourceAttrs int
	switch inputs.NumAttrs {
	case AttrsNone:
		numResourceAttrs = 0
	case AttrsOne:
		numResourceAttrs = 1
	case AttrsTwo:
		numResourceAttrs = 2
	}

	for i := 0; i < numResourceAttrs; i++ {
		attrs.Insert("my-name", pdata.NewAttributeValueString("resourceName"))
	}

	ilmSlice := rm.InstrumentationLibraryMetrics()
	ilmSlice.Resize(1)
	ilm := ilmSlice.At(0)
	il := ilm.InstrumentationLibrary()
	il.InitEmpty()
	il.SetName("my-il-name")
	il.SetVersion("my-il-version")

	metricSlice := ilm.Metrics()
	metricSlice.Resize(1)
	metric := metricSlice.At(0)
	metric.InitEmpty()

	mDesc := metric.MetricDescriptor()
	mDesc.InitEmpty()
	mDesc.SetName(fmt.Sprintf("metric-%d", metricCounter))
	mDesc.SetDescription("my-description")
	metricCounter++

	mDesc.SetUnit("x")

	numPts := 0
	switch inputs.NumPtsPerMetric {
	case NumPtsPerMetricOne:
		numPts = 1
	case NumPtsPerMetricMany:
		numPts = 1024
	default:
		log.Error("invalid value for inputs.NumPtsPerMetric: " + inputs.NumPtsPerMetric)
	}

	startTime := uint64(42)
	stepSize := uint64(100)

	switch inputs.MetricType {
	case MetricTypeMonotonicDouble:
		mDesc.SetType(pdata.MetricTypeMonotonicDouble)
		populateDoublePts(metric, numPts, startTime, stepSize, inputs.NumLabels)
	case MetricTypeDouble:
		mDesc.SetType(pdata.MetricTypeDouble)
		populateDoublePts(metric, numPts, startTime, stepSize, inputs.NumLabels)
	case MetricTypeMonotonicInt:
		mDesc.SetType(pdata.MetricTypeMonotonicInt64)
		populateIntPts(metric, numPts, startTime, stepSize, inputs.NumLabels)
	case MetricTypeInt:
		mDesc.SetType(pdata.MetricTypeInt64)
		populateIntPts(metric, numPts, startTime, stepSize, inputs.NumLabels)
	case MetricTypeSummary:
		mDesc.SetType(pdata.MetricTypeSummary)
		pts := metric.SummaryDataPoints()
		pts.Resize(numPts)
		for i := 0; i < numPts; i++ {
			pt := pts.At(i)
			pt.InitEmpty()
			pt.SetStartTime(pdata.TimestampUnixNano(startTime))
			pt.SetTimestamp(getTimestamp(startTime, stepSize, 1))
			pt.SetCount(uint64(10 + i))
			pt.SetSum(float64(100 + i))
			insertLabels(pt.LabelsMap(), inputs.NumLabels)
		}
	case MetricTypeHistogram:
		mDesc.SetType(pdata.MetricTypeHistogram)
		pts := metric.HistogramDataPoints()
		pts.Resize(numPts)
		for i := 0; i < numPts; i++ {
			pt := pts.At(i)
			pt.SetStartTime(pdata.TimestampUnixNano(startTime))
			ts := getTimestamp(startTime, stepSize, i)
			pt.SetTimestamp(ts)
			insertLabels(pt.LabelsMap(), inputs.NumLabels)
			buckets := pt.Buckets()
			numBuckets := 4
			buckets.Resize(numBuckets)
			for j := 0; j < numBuckets; j++ {
				bucket := buckets.At(j)
				bucket.SetCount(uint64(j))
				ex := bucket.Exemplar()
				ex.InitEmpty()
				ex.SetTimestamp(ts)
				ex.SetValue(1.0)
			}
			ptCounter++
		}
	}
	return md
}

func populateIntPts(metric pdata.Metric, numPts int, startTime uint64, stepSize uint64, numLabels PICTNumLabels) {
	pts := metric.Int64DataPoints()
	pts.Resize(numPts)
	for i := 0; i < numPts; i++ {
		pt := pts.At(i)
		pt.SetValue(int64(ptCounter))
		pt.SetStartTime(pdata.TimestampUnixNano(startTime))
		pt.SetTimestamp(getTimestamp(startTime, stepSize, i))
		insertLabels(pt.LabelsMap(), numLabels)
		ptCounter++
	}
}

func populateDoublePts(metric pdata.Metric, numPts int, startTime uint64, stepSize uint64, numLabels PICTNumLabels) {
	pts := metric.DoubleDataPoints()
	pts.Resize(numPts)
	for i := 0; i < numPts; i++ {
		pt := pts.At(i)
		pt.SetValue(float64(i))
		pt.SetStartTime(pdata.TimestampUnixNano(startTime))
		pt.SetTimestamp(getTimestamp(startTime, stepSize, i))
		insertLabels(pt.LabelsMap(), numLabels)
		ptCounter++
	}
}

func insertLabels(lm pdata.StringMap, pictNumLabels PICTNumLabels) {
	numLabels := 0
	switch pictNumLabels {
	case LabelsNone:
	case LabelsOne:
		numLabels = 1
	case LabelsMany:
		numLabels = 16
	default:
		log.Error("unrecognized value for PICTNumLabels: " + pictNumLabels)
	}
	for i := 0; i < numLabels; i++ {
		k := fmt.Sprintf("label-key-%d", i)
		v := fmt.Sprintf("label-value-%d", i)
		lm.Insert(k, v)
	}
}

func getTimestamp(startTime uint64, stepSize uint64, i int) pdata.TimestampUnixNano {
	return pdata.TimestampUnixNano(startTime + (stepSize * uint64(i)))
}
