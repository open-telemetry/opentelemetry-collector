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

// // Start of PICT inputs for generating golden dataset metrics (pict_input_metrics.txt) ////

// PICTMetricInputs defines one pairwise combination of MetricData variations
type PICTMetricInputs struct {
	// Specifies the number of points on each metric.
	NumPtsPerMetric PICTNumPtsPerMetric
	// Specifies the types of metrics that can be generated.
	MetricType PICTMetricType
	// Specifies the number of labels on each datapoint.
	NumPtLabels PICTNumPtLabels
	// Specifies the number of attributes on each resource.
	NumResourceAttrs PICTNumResourceAttrs
}

// Enumerates the types of metrics that can be generated.
type PICTMetricType string

const (
	MetricTypeInt             PICTMetricType = "Int"
	MetricTypeMonotonicInt    PICTMetricType = "MonotonicInt"
	MetricTypeDouble          PICTMetricType = "Double"
	MetricTypeMonotonicDouble PICTMetricType = "MonotonicDouble"
	MetricTypeHistogram       PICTMetricType = "Histogram"
	MetricTypeSummary         PICTMetricType = "Summary"
)

// Enumerates the number of labels on each datapoint.
type PICTNumPtLabels string

const (
	LabelsNone PICTNumPtLabels = "NoLabels"
	LabelsOne  PICTNumPtLabels = "OneLabel"
	LabelsMany PICTNumPtLabels = "ManyLabels"
)

// Enumerates the number of points on each metric.
type PICTNumPtsPerMetric string

const (
	NumPtsPerMetricOne  PICTNumPtsPerMetric = "OnePt"
	NumPtsPerMetricMany PICTNumPtsPerMetric = "ManyPts"
)

// Enumerates the number of attributes on each resource.
type PICTNumResourceAttrs string

const (
	AttrsNone PICTNumResourceAttrs = "NoAttrs"
	AttrsOne  PICTNumResourceAttrs = "OneAttr"
	AttrsTwo  PICTNumResourceAttrs = "TwoAttrs"
)
