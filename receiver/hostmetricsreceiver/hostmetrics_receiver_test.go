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

package hostmetricsreceiver

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
)

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr: &cpuscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
				ReportPerCPU:   true,
			},
		},
	}

	factories := map[string]internal.Factory{
		cpuscraper.TypeStr: &cpuscraper.Factory{},
	}

	receiver, err := NewHostMetricsReceiver(context.Background(), zap.NewNop(), config, factories, sink)

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertMetricData(t, got)
		return true
	}, time.Second, 10*time.Millisecond, "No metrics were collected")
}

func assertMetricData(t *testing.T, got []pdata.Metrics) {
	metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

	// expect 1 metric
	assert.Equal(t, 1, metrics.Len())

	// for cpu seconds metric, expect a datapoint for each state label & core combination with at least 4 standard states
	hostCPUTimeMetric := metrics.At(0)
	internal.AssertDescriptorEqual(t, cpuscraper.MetricCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
	assert.GreaterOrEqual(t, hostCPUTimeMetric.Int64DataPoints().Len(), runtime.NumCPU()*4)
	internal.AssertInt64MetricLabelExists(t, hostCPUTimeMetric, 0, cpuscraper.CPULabel)
	internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, cpuscraper.StateLabel, cpuscraper.UserStateLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, cpuscraper.StateLabel, cpuscraper.SystemStateLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, cpuscraper.StateLabel, cpuscraper.IdleStateLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, cpuscraper.StateLabel, cpuscraper.InterruptStateLabelValue)
}
