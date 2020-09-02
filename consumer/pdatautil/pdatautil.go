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
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// MetricsToMetricsData returns the `[]consumerdata.MetricsData` representation of the `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsToMetricsData(md pdata.Metrics) []consumerdata.MetricsData {
	if imd, ok := md.InternalOpaque.(data.MetricData); ok {
		return internaldata.MetricsToOC(imd)
	}
	panic("Unsupported metrics type.")
}

// MetricsFromMetricsData returns the `pdata.Metrics` representation of the `[]consumerdata.MetricsData`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsFromMetricsData(ocmds []consumerdata.MetricsData) pdata.Metrics {
	return MetricsFromInternalMetrics(internaldata.OCSliceToMetrics(ocmds))
}

// MetricsToInternalMetrics returns the `data.MetricData` representation of the `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsToInternalMetrics(md pdata.Metrics) data.MetricData {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return ims
	}
	panic("Unsupported metrics type.")
}

// MetricsFromInternalMetrics returns the `pdata.Metrics` representation of the `data.MetricData`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsFromInternalMetrics(md data.MetricData) pdata.Metrics {
	return pdata.Metrics{InternalOpaque: md}
}

// CloneMetrics returns a clone of the given `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func CloneMetrics(md pdata.Metrics) pdata.Metrics {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return pdata.Metrics{InternalOpaque: ims.Clone()}
	}
	panic("Unsupported metrics type.")
}

func MetricCount(md pdata.Metrics) int {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return ims.MetricCount()
	}
	panic("Unsupported metrics type.")
}

func MetricAndDataPointCount(md pdata.Metrics) (int, int) {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return ims.MetricAndDataPointCount()
	}
	panic("Unsupported metrics type.")
}

func MetricPointCount(md pdata.Metrics) int {
	_, points := MetricAndDataPointCount(md)
	return points
}
