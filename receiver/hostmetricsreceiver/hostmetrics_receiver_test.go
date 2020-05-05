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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
)

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr: &cpuscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
				ReportPerCPU:   true,
			},
			memoryscraper.TypeStr: &memoryscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
			},
		},
	}

	factories := map[string]internal.Factory{
		cpuscraper.TypeStr:    &cpuscraper.Factory{},
		memoryscraper.TypeStr: &memoryscraper.Factory{},
	}

	receiver, err := NewHostMetricsReceiver(context.Background(), zap.NewNop(), config, factories, sink)

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	time.Sleep(180 * time.Millisecond)

	got := sink.AllMetrics()

	// expect 2 MetricData objects
	assert.Equal(t, 2, len(got))

	// extract the names of all returned metrics
	metricNames := make(map[string]bool)
	for _, metricData := range got {
		metrics := internal.AssertMetricDataAndGetMetricsSlice(t, metricData)
		for i := 0; i < metrics.Len(); i++ {
			metricNames[metrics.At(i).MetricDescriptor().Name()] = true
		}
	}

	// expect 2 metrics
	assert.Equal(t, 2, len(metricNames))

	// expected metric names
	assert.Contains(t, metricNames, "host/cpu/time")
	assert.Contains(t, metricNames, "host/memory/used")
}
