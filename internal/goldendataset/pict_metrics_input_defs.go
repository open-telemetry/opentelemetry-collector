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

type PICTMetricType string

type PICTMetricInputs struct {
	NumPtsPerMetric PICTNumPtsPerMetric
	MetricType      PICTMetricType
	NumLabels       PICTNumPtLabels
	NumAttrs        PICTNumResrouceAttrs
}

const (
	MetricTypeInt             PICTMetricType = "Int"
	MetricTypeMonotonicInt    PICTMetricType = "MonotonicInt"
	MetricTypeDouble          PICTMetricType = "Double"
	MetricTypeMonotonicDouble PICTMetricType = "MonotonicDouble"
	MetricTypeHistogram       PICTMetricType = "Histogram"
	MetricTypeSummary         PICTMetricType = "Summary"
)

type PICTNumPtLabels string

const (
	LabelsNone PICTNumPtLabels = "NoLabels"
	LabelsOne  PICTNumPtLabels = "OneLabel"
	LabelsMany PICTNumPtLabels = "ManyLabels"
)

type PICTNumPtsPerMetric string

const (
	NumPtsPerMetricOne  PICTNumPtsPerMetric = "OnePt"
	NumPtsPerMetricMany PICTNumPtsPerMetric = "ManyPts"
)

type PICTNumResrouceAttrs string

const (
	AttrsNone PICTNumResrouceAttrs = "NoAttrs"
	AttrsOne  PICTNumResrouceAttrs = "OneAttr"
	AttrsTwo  PICTNumResrouceAttrs = "TwoAttrs"
)
