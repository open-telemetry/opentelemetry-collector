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

package internal

import (
	"strconv"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"

	"go.opentelemetry.io/collector/model/pdata"
)

func isUsefulLabelPdata(mType pdata.MetricDataType, labelKey string) bool {
	switch labelKey {
	case model.MetricNameLabel, model.InstanceLabel, model.SchemeLabel, model.MetricsPathLabel, model.JobLabel:
		return false
	case model.BucketLabel:
		return mType != pdata.MetricDataTypeIntHistogram &&
			mType != pdata.MetricDataTypeHistogram
	case model.QuantileLabel:
		return mType != pdata.MetricDataTypeSummary
	}
	return true
}

func getBoundaryPdata(metricType pdata.MetricDataType, labels labels.Labels) (float64, error) {
	labelName := ""
	switch metricType {
	case pdata.MetricDataTypeHistogram, pdata.MetricDataTypeIntHistogram:
		labelName = model.BucketLabel
	case pdata.MetricDataTypeSummary:
		labelName = model.QuantileLabel
	default:
		return 0, errNoBoundaryLabel
	}

	v := labels.Get(labelName)
	if v == "" {
		return 0, errEmptyBoundaryLabel
	}

	return strconv.ParseFloat(v, 64)
}

func convToPdataMetricType(metricType textparse.MetricType) pdata.MetricDataType {
	switch metricType {
	case textparse.MetricTypeCounter:
		// always use float64, as it's the internal data type used in prometheus
		return pdata.MetricDataTypeSum
	// textparse.MetricTypeUnknown is converted to gauge by default to fix Prometheus untyped metrics from being dropped
	case textparse.MetricTypeGauge, textparse.MetricTypeUnknown:
		return pdata.MetricDataTypeGauge
	case textparse.MetricTypeHistogram:
		return pdata.MetricDataTypeHistogram
	// dropping support for gaugehistogram for now until we have an official spec of its implementation
	// a draft can be found in: https://docs.google.com/document/d/1KwV0mAXwwbvvifBvDKH_LU1YjyXE_wxCkHNoCGq1GX0/edit#heading=h.1cvzqd4ksd23
	// case textparse.MetricTypeGaugeHistogram:
	//	return metricspb.MetricDescriptor_GAUGE_DISTRIBUTION
	case textparse.MetricTypeSummary:
		return pdata.MetricDataTypeSummary
	default:
		// including: textparse.MetricTypeInfo, textparse.MetricTypeStateset
		return pdata.MetricDataTypeNone
	}
}
