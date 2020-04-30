// Copyright 2020, OpenTelemetry Authors
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

package cpuscraper

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, []pdata.Metrics)

func TestScrapeMetrics_MinimalData(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 2 metrics
		assert.Equal(t, 2, metrics.Len())

		// for cpu seconds metric, expect 4 timeseries with appropriate labels
		hostCPUTimeMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, MetricCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
		assert.Equal(t, 4, hostCPUTimeMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelDoesNotExist(t, hostCPUTimeMetric, 0, CPULabel)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, StateLabel, UserStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, StateLabel, SystemStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, StateLabel, IdleStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, StateLabel, InterruptStateLabelValue)

		// for cpu utilization metric, expect 1 timeseries with a value < 100
		hostCPUUtilizationMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, MetricCPUUtilizationDescriptor, hostCPUUtilizationMetric.MetricDescriptor())
		assert.Equal(t, 1, hostCPUUtilizationMetric.DoubleDataPoints().Len())
		assert.LessOrEqual(t, hostCPUUtilizationMetric.DoubleDataPoints().At(0).Value(), float64(100))
	})
}

func TestScrapeMetrics_AllData(t *testing.T) {
	config := &Config{
		ReportPerCPU: true,
	}

	createScraperAndValidateScrapedMetrics(t, config, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 2 metris
		assert.Equal(t, 2, metrics.Len())

		// for cpu seconds metric, expect 4*cores timeseries with appropriate labels
		hostCPUTimeMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, MetricCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
		assert.Equal(t, 4*runtime.NumCPU(), hostCPUTimeMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelExists(t, hostCPUTimeMetric, 0, CPULabel)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, StateLabel, UserStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, StateLabel, SystemStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, StateLabel, IdleStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, StateLabel, InterruptStateLabelValue)

		// for cpu utilization metric, expect #cores timeseries each with a value <= 100
		hostCPUUtilizationMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, MetricCPUUtilizationDescriptor, hostCPUUtilizationMetric.MetricDescriptor())
		ddp := hostCPUUtilizationMetric.DoubleDataPoints()
		assert.Equal(t, runtime.NumCPU(), ddp.Len())
		for i := 0; i < ddp.Len(); i++ {
			assert.LessOrEqual(t, ddp.At(i).Value(), float64(100))
		}
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	config.SetCollectionInterval(100 * time.Millisecond)

	sink := &exportertest.SinkMetricsExporter{}

	scraper, err := NewCPUScraper(context.Background(), config, sink)
	require.NoError(t, err, "Failed to create cpu scraper: %v", err)

	err = scraper.Start(context.Background())
	require.NoError(t, err, "Failed to start cpu scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertFn(t, got)
		return true
	}, time.Second, 10*time.Millisecond, "No metrics were collected")
}
