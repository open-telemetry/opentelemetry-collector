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
	"github.com/stretchr/testify/require"
	googleproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestMetricCount(t *testing.T) {
	metrics := testdata.GenerateMetricsTwoMetrics()
	assert.Equal(t, 2, MetricCount(metrics))

	metrics = MetricsFromMetricsData([]consumerdata.MetricsData{
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
	})
	assert.Equal(t, 1, MetricCount(metrics))
}

func TestMetricAndDataPointCount(t *testing.T) {
	metrics := testdata.GenerateMetricsOneMetric()
	metricsCount, dataPointsCount := MetricAndDataPointCount(metrics)
	assert.Equal(t, 1, metricsCount)
	assert.Equal(t, 2, dataPointsCount)

	metrics = MetricsFromMetricsData([]consumerdata.MetricsData{
		{
			Metrics: []*ocmetrics.Metric{
				{
					MetricDescriptor: &ocmetrics.MetricDescriptor{
						Name: "gauge",
						Type: ocmetrics.MetricDescriptor_GAUGE_INT64,
					},
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
	})
	metricsCount, dataPointsCount = MetricAndDataPointCount(metrics)
	assert.Equal(t, 1, metricsCount)
	assert.Equal(t, 3, dataPointsCount)
}

func TestMetricConvertInternalToFromOC(t *testing.T) {
	metrics := testdata.GenerateMetricsOneMetric()
	mdOc := MetricsToMetricsData(metrics)
	assert.Len(t, mdOc, 1)
	assert.Len(t, mdOc[0].Metrics, 1)
	oldMetrics := MetricsFromMetricsData(mdOc)
	assert.Equal(t, 1, MetricCount(oldMetrics))
	mdNew := oldMetrics
	assert.Equal(t, 1, mdNew.MetricCount())
}

func TestMetricCloneInternal(t *testing.T) {
	metrics := testdata.GenerateMetricsOneMetric()
	clone := CloneMetrics(metrics)
	assert.EqualValues(t, metrics, clone)
}

func TestMetricCloneOc(t *testing.T) {
	oc := MetricsToMetricsData(testdata.GenerateMetricsOneMetric())
	require.Len(t, oc, 1)
	require.Len(t, oc[0].Metrics, 1)
	metrics := MetricsFromMetricsData(oc)
	metricsClone := CloneMetrics(metrics)
	ocClone := MetricsToMetricsData(metricsClone)
	require.Len(t, ocClone, 1)
	require.Len(t, ocClone[0].Metrics, 1)
	assert.True(t, googleproto.Equal(oc[0].Node, ocClone[0].Node))
	assert.True(t, googleproto.Equal(oc[0].Resource, ocClone[0].Resource))
	assert.True(t, googleproto.Equal(oc[0].Metrics[0], ocClone[0].Metrics[0]))
}
