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
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
)

var counter int64

func GenerateMetrics(metricPairsFile string) ([]data.MetricData, error) {
	pictData, err := loadPictOutputFile(metricPairsFile)
	if err != nil {
		return nil, err
	}
	var out []data.MetricData
	for index, values := range pictData {
		if index == 0 {
			continue
		}
		metricInputs := PICTMetricInputs{
			MetricType:       PICTMetricType(values[0]),
			NumResourceAttrs: PICTNumResourceAttrs(values[1]),
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

	var count int
	switch inputs.NumResourceAttrs {
	case AttrsNone:
		count = 0
	case AttrsOne:
		count = 1
	case AttrsTwo:
		count = 2
	}

	for i := 0; i < count; i++ {
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
	mDesc.SetName("my-md-name")
	mDesc.SetDescription("my-md-desc")
	mDesc.SetUnit("x")

	numPts := 10

	startTime := uint64(42)
	stepSize := uint64(100)

	switch inputs.MetricType {
	case MetricTypeDouble:
		mDesc.SetType(pdata.MetricTypeDouble)
		ptSlice := metric.DoubleDataPoints()
		ptSlice.Resize(numPts)
		for i := 0; i < numPts; i++ {
			pt := ptSlice.At(i)
			pt.SetValue(float64(i))
			lm := pt.LabelsMap()
			lm.Insert("point-label-key", "point-label-value")
			counter++
		}
	case MetricTypeInt:
		mDesc.SetType(pdata.MetricTypeInt64)
		ptSlice := metric.Int64DataPoints()
		ptSlice.Resize(numPts)
		for i := 0; i < numPts; i++ {
			pt := ptSlice.At(i)
			pt.SetValue(counter)
			ptTime := startTime + (stepSize * uint64(i))
			pt.SetStartTime(pdata.TimestampUnixNano(ptTime))
			lm := pt.LabelsMap()
			lm.Insert("point-label-key", "point-label-value")
			counter++
		}
	}
	return md
}
