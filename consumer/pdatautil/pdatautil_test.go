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
	"testing"

	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold/testdataold"
)

func TestMetricCount(t *testing.T) {
	metrics := pdata.Metrics{InternalOpaque: testdataold.GenerateMetricDataTwoMetrics()}
	assert.Equal(t, 2, MetricCount(metrics))

	metrics = pdata.Metrics{InternalOpaque: []consumerdata.MetricsData{
		{
			Metrics: []*ocmetrics.Metric{
				{
					Timeseries: []*ocmetrics.TimeSeries{
						{
							Points: []*ocmetrics.Point{
								{Value: &ocmetrics.Point_Int64Value{Int64Value: 123}},
							},
						},
					},
				},
			},
		},
	}}
	assert.Equal(t, 1, MetricCount(metrics))
}

func TestMetricAndDataPointCount(t *testing.T) {
	metrics := pdata.Metrics{InternalOpaque: testdataold.GenerateMetricDataTwoMetrics()}
	metricsCount, dataPointsCount := MetricAndDataPointCount(metrics)
	assert.Equal(t, 2, metricsCount)
	assert.Equal(t, 4, dataPointsCount)

	metrics = pdata.Metrics{InternalOpaque: []consumerdata.MetricsData{
		{
			Metrics: []*ocmetrics.Metric{
				{
					Timeseries: []*ocmetrics.TimeSeries{
						{
							Points: []*ocmetrics.Point{
								{Value: &ocmetrics.Point_Int64Value{Int64Value: 123}},
								{Value: &ocmetrics.Point_Int64Value{Int64Value: 345}},
								{Value: &ocmetrics.Point_Int64Value{Int64Value: 567}},
							},
						},
					},
				},
			},
		},
	}}
	metricsCount, dataPointsCount = MetricAndDataPointCount(metrics)
	assert.Equal(t, 1, metricsCount)
	assert.Equal(t, 3, dataPointsCount)
}
