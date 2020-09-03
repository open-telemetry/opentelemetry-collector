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
)

// GenerateMetricDatas takes the filename of a PICT-generated file, walks through all of the rows in the PICT
// file and for each row, generates a MetricData object, collecting them and returning them to the caller.
func GenerateMetricDatas(metricPairsFile string) ([]pdata.Metrics, error) {
	pictData, err := loadPictOutputFile(metricPairsFile)
	if err != nil {
		return nil, err
	}
	var out []pdata.Metrics
	for i, values := range pictData {
		if i == 0 {
			continue
		}
		metricInputs := PICTMetricInputs{
			NumPtsPerMetric: PICTNumPtsPerMetric(values[0]),
			MetricType:      PICTMetricDataType(values[1]),
			NumPtLabels:     PICTNumPtLabels(values[2]),
		}
		cfg := pictToCfg(metricInputs)
		cfg.MetricNamePrefix = fmt.Sprintf("pict_%d_", i)
		md := MetricDataFromCfg(cfg)
		out = append(out, md)
	}
	return out, nil
}

func pictToCfg(inputs PICTMetricInputs) MetricCfg {
	cfg := DefaultCfg()
	switch inputs.NumResourceAttrs {
	case AttrsNone:
		cfg.NumResourceAttrs = 0
	case AttrsOne:
		cfg.NumResourceAttrs = 1
	case AttrsTwo:
		cfg.NumResourceAttrs = 2
	}

	switch inputs.NumPtsPerMetric {
	case NumPtsPerMetricOne:
		cfg.NumPtsPerMetric = 1
	case NumPtsPerMetricMany:
		cfg.NumPtsPerMetric = 16
	}

	switch inputs.MetricType {
	case MetricTypeIntGauge:
		cfg.MetricDescriptorType = pdata.MetricDataTypeIntGauge
	case MetricTypeMonotonicIntSum:
		cfg.MetricDescriptorType = pdata.MetricDataTypeIntSum
		cfg.IsMonotonicSum = true
	case MetricTypeNonMonotonicIntSum:
		cfg.MetricDescriptorType = pdata.MetricDataTypeIntSum
		cfg.IsMonotonicSum = false
	case MetricTypeDoubleGauge:
		cfg.MetricDescriptorType = pdata.MetricDataTypeDoubleGauge
	case MetricTypeMonotonicDoubleSum:
		cfg.MetricDescriptorType = pdata.MetricDataTypeDoubleSum
		cfg.IsMonotonicSum = true
	case MetricTypeNonMonotonicDoubleSum:
		cfg.MetricDescriptorType = pdata.MetricDataTypeDoubleSum
		cfg.IsMonotonicSum = false
	case MetricTypeIntHistogram:
		cfg.MetricDescriptorType = pdata.MetricDataTypeIntHistogram
	case MetricTypeDoubleHistogram:
		cfg.MetricDescriptorType = pdata.MetricDataTypeDoubleHistogram
	default:
		panic("Should not happen, unsupported type " + string(inputs.MetricType))
	}

	switch inputs.NumPtLabels {
	case LabelsNone:
		cfg.NumPtLabels = 0
	case LabelsOne:
		cfg.NumPtLabels = 1
	case LabelsMany:
		cfg.NumPtLabels = 16
	}
	return cfg
}
