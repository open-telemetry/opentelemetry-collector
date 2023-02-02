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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

// MetricType specifies the type of data in a Metric.
type MetricType int32

const (
	// MetricTypeEmpty means that metric type is unset.
	MetricTypeEmpty MetricType = iota
	MetricTypeGauge
	MetricTypeSum
	MetricTypeHistogram
	MetricTypeExponentialHistogram
	MetricTypeSummary
)

// String returns the string representation of the MetricType.
func (mdt MetricType) String() string {
	switch mdt {
	case MetricTypeEmpty:
		return "Empty"
	case MetricTypeGauge:
		return "Gauge"
	case MetricTypeSum:
		return "Sum"
	case MetricTypeHistogram:
		return "Histogram"
	case MetricTypeExponentialHistogram:
		return "ExponentialHistogram"
	case MetricTypeSummary:
		return "Summary"
	}
	return ""
}
