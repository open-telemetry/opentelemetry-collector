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

func GenerateMetrics(metricInputs *PICTMetricInputs) (data.MetricData, error) {
	md := data.NewMetricData()

	rmSlice := md.ResourceMetrics()
	rmSlice.Resize(1)

	rm := rmSlice.At(0)

	rsrc := rm.Resource()
	rsrc.InitEmpty()
	attrs := rsrc.Attributes()

	var count int
	switch metricInputs.MetricInputs {
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
	mDesc.SetType(pdata.MetricTypeInt64)
	mDesc.SetUnit("s")
	ptSlice := metric.Int64DataPoints()
	ptSlice.Resize(1)
	pt := ptSlice.At(0)
	pt.SetValue(42)
	pt.SetStartTime(pdata.TimestampUnixNano(uint64(128)))
	lm := pt.LabelsMap()
	lm.Insert("lblKey", "lblVal")
	return md, nil
}
