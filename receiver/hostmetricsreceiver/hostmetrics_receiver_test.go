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

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/diskscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/filesystemscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/networkscraper"
)

var standardMetrics = []string{
	"host/cpu/time",
	"host/memory/used",
	"host/disk/bytes",
	"host/disk/ops",
	"host/disk/time",
	"host/filesystem/used",
	"host/network/packets",
	"host/network/dropped_packets",
	"host/network/errors",
	"host/network/bytes",
	"host/network/tcp_connections",
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
		CollectionInterval: 100 * time.Millisecond,
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:        &cpuscraper.Config{ReportPerCPU: true},
			diskscraper.TypeStr:       &diskscraper.Config{},
			filesystemscraper.TypeStr: &filesystemscraper.Config{},
			memoryscraper.TypeStr:     &memoryscraper.Config{},
			networkscraper.TypeStr:    &networkscraper.Config{},
		},
	}

	factories := map[string]internal.Factory{
		cpuscraper.TypeStr:        &cpuscraper.Factory{},
		diskscraper.TypeStr:       &diskscraper.Factory{},
		filesystemscraper.TypeStr: &filesystemscraper.Factory{},
		memoryscraper.TypeStr:     &memoryscraper.Factory{},
		networkscraper.TypeStr:    &networkscraper.Factory{},
	}

	receiver, err := newHostMetricsReceiver(context.Background(), zap.NewNop(), config, factories, sink)

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	ctx, cancelFn := context.WithCancel(context.Background())
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	// canceling the context provided to Start should not cancel any async processes initiated by the receiver
	cancelFn()

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertIncludesAllMetrics(t, got[0])
		return true
	}, time.Second, 10*time.Millisecond, "No metrics were collected after 1s")
}

func assertIncludesAllMetrics(t *testing.T, got pdata.Metrics) {
	metrics := assertMetricDataAndGetMetricsSlice(t, got)

	// extract the names of all returned metrics
	metricNames := make(map[string]bool)
	for i := 0; i < metrics.Len(); i++ {
		metricNames[metrics.At(i).MetricDescriptor().Name()] = true
	}

	// the expected list of metrics returned is os dependent
	expectedMetrics := append(standardMetrics, systemSpecificMetrics[runtime.GOOS]...)
	assert.Equal(t, len(expectedMetrics), len(metricNames))
	for _, expected := range expectedMetrics {
		assert.Contains(t, metricNames, expected)
	}
}

func assertMetricDataAndGetMetricsSlice(t *testing.T, metrics pdata.Metrics) pdata.MetricSlice {
	md := pdatautil.MetricsToInternalMetrics(metrics)

	// expect 1 ResourceMetrics object
	rms := md.ResourceMetrics()
	assert.Equal(t, 1, rms.Len())
	rm := rms.At(0)

	// expect 1 InstrumentationLibraryMetrics object
	ilms := rm.InstrumentationLibraryMetrics()
	assert.Equal(t, 1, ilms.Len())
	return ilms.At(0).Metrics()
}
