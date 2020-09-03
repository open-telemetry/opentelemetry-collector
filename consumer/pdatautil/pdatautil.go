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

// Package pdatautil is a temporary package to allow components to transition to the new API.
// It will be removed when pdata.Metrics will be finalized.
package pdatautil

import (
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// MetricsToMetricsData returns the `[]consumerdata.MetricsData` representation of the `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsToMetricsData(md pdata.Metrics) []consumerdata.MetricsData {
	return internaldata.MetricsToOC(md)
}

// MetricsFromMetricsData returns the `pdata.Metrics` representation of the `[]consumerdata.MetricsData`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsFromMetricsData(ocmds []consumerdata.MetricsData) pdata.Metrics {
	return internaldata.OCSliceToMetrics(ocmds)
}

// CloneMetrics returns a clone of the given `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func CloneMetrics(md pdata.Metrics) pdata.Metrics {
	return md.Clone()
}

func MetricCount(md pdata.Metrics) int {
	return md.MetricCount()
}

func MetricAndDataPointCount(md pdata.Metrics) (int, int) {
	return md.MetricAndDataPointCount()
}

func MetricPointCount(md pdata.Metrics) int {
	_, points := md.MetricAndDataPointCount()
	return points
}
