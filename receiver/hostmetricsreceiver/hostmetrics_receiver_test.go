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
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/diskscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/filesystemscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
)

var standardMetrics = []string{
	"host/cpu/time",
	"host/memory/used",
	"host/disk/bytes",
	"host/disk/ops",
	"host/disk/time",
	"host/filesystem/used",
}

var systemSpecificMetrics = map[string][]string{
	"linux":   {"host/filesystem/inodes/used"},
	"darwin":  {"host/filesystem/inodes/used"},
	"freebsd": {"host/filesystem/inodes/used"},
	"openbsd": {"host/filesystem/inodes/used"},
	"solaris": {"host/filesystem/inodes/used"},
}

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr: &cpuscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
				ReportPerCPU:   true,
			},
			diskscraper.TypeStr: &diskscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
			},
			filesystemscraper.TypeStr: &filesystemscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
			},
			memoryscraper.TypeStr: &memoryscraper.Config{
				ConfigSettings: internal.ConfigSettings{CollectionIntervalValue: 100 * time.Millisecond},
			},
		},
	}

	factories := map[string]internal.Factory{
		cpuscraper.TypeStr:        &cpuscraper.Factory{},
		diskscraper.TypeStr:       &diskscraper.Factory{},
		filesystemscraper.TypeStr: &filesystemscraper.Factory{},
		memoryscraper.TypeStr:     &memoryscraper.Factory{},
	}

	receiver, err := NewHostMetricsReceiver(context.Background(), zap.NewNop(), config, factories, sink)

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	time.Sleep(180 * time.Millisecond)

	got := sink.AllMetrics()

	// expect a MetricData object for each configured scraper
	assert.Equal(t, len(config.Scrapers), len(got))

	// extract the names of all returned metrics
	metricNames := make(map[string]bool)
	for _, metricData := range got {
		metrics := internal.AssertMetricDataAndGetMetricsSlice(t, metricData)
		for i := 0; i < metrics.Len(); i++ {
			metricNames[metrics.At(i).MetricDescriptor().Name()] = true
		}
	}

	// the expected list of metrics returned is os dependent
	expectedMetrics := append(standardMetrics, systemSpecificMetrics[runtime.GOOS]...)
	assert.Equal(t, len(expectedMetrics), len(metricNames))
	for _, expected := range expectedMetrics {
		assert.Contains(t, metricNames, expected)
	}
}
