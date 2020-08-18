// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	googleproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// MetricsToMetricsData returns the `[]consumerdata.MetricsData` representation of the `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsToMetricsData(md pdata.Metrics) []consumerdata.MetricsData {
	if cmd, ok := md.InternalOpaque.([]consumerdata.MetricsData); ok {
		return cmd
	}
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return internaldata.MetricDataToOC(ims)
	}
	panic("Unsupported metrics type.")
}

// MetricsFromMetricsData returns the `pdata.Metrics` representation of the `[]consumerdata.MetricsData`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsFromMetricsData(ocmds []consumerdata.MetricsData) pdata.Metrics {
	return pdata.Metrics{InternalOpaque: ocmds}
}

// MetricsToInternalMetrics returns the `data.MetricData` representation of the `pdata.Metrics`.
//
// This is a temporary function that will be removed when the new internal pdata.Metrics will be finalized.
func MetricsToInternalMetrics(md pdata.Metrics) data.MetricData {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return ims
	}
	if cmd, ok := md.InternalOpaque.([]consumerdata.MetricsData); ok {
		return internaldata.OCSliceToMetricData(cmd)
	}
	panic("Unsupported metrics type.")
}

// MetricsFromMetricsData returns the `pdata.Metrics` representation of the `data.MetricData`.
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
	if ocmds, ok := md.InternalOpaque.([]consumerdata.MetricsData); ok {
		clone := make([]consumerdata.MetricsData, 0, len(ocmds))
		for _, ocmd := range ocmds {
			clone = append(clone, CloneMetricsDataOld(ocmd))
		}
		return pdata.Metrics{InternalOpaque: clone}
	}
	panic("Unsupported metrics type.")
}

func MetricCount(md pdata.Metrics) int {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return ims.MetricCount()
	}
	if ocmds, ok := md.InternalOpaque.([]consumerdata.MetricsData); ok {
		metricCount := 0
		for _, ocmd := range ocmds {
			metricCount += len(ocmd.Metrics)
		}
		return metricCount
	}
	panic("Unsupported metrics type.")
}

func MetricAndDataPointCount(md pdata.Metrics) (int, int) {
	if ims, ok := md.InternalOpaque.(data.MetricData); ok {
		return ims.MetricAndDataPointCount()
	}
	if ocmds, ok := md.InternalOpaque.([]consumerdata.MetricsData); ok {
		metricCount := 0
		dataPointCount := 0
		for _, ocmd := range ocmds {
			mc, dpc := TimeseriesAndPointCount(ocmd)
			metricCount += mc
			dataPointCount += dpc
		}
		return metricCount, dataPointCount
	}
	panic("Unsupported metrics type.")
}

func MetricPointCount(md pdata.Metrics) int {
	_, points := MetricAndDataPointCount(md)
	return points
}

// CloneMetricsDataOld copied from processors.cloneMetricsDataOld
func CloneMetricsDataOld(md consumerdata.MetricsData) consumerdata.MetricsData {
	clone := consumerdata.MetricsData{
		Node:     googleproto.Clone(md.Node).(*commonpb.Node),
		Resource: googleproto.Clone(md.Resource).(*resourcepb.Resource),
	}

	if md.Metrics != nil {
		clone.Metrics = make([]*metricspb.Metric, 0, len(md.Metrics))

		for _, metric := range md.Metrics {
			metricClone := googleproto.Clone(metric).(*metricspb.Metric)
			clone.Metrics = append(clone.Metrics, metricClone)
		}
	}

	return clone
}

// TimeseriesAndPointCount copied from exporterhelper.measureMetricsExport
func TimeseriesAndPointCount(md consumerdata.MetricsData) (int, int) {
	numTimeSeries := 0
	numPoints := 0
	for _, metric := range md.Metrics {
		tss := metric.GetTimeseries()
		numTimeSeries += len(metric.GetTimeseries())
		for _, ts := range tss {
			numPoints += len(ts.GetPoints())
		}
	}
	return numTimeSeries, numPoints
}
