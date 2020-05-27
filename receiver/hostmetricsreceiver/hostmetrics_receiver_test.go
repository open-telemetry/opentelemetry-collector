// Copyright The OpenTelemetry Authors
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
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/loadscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/networkscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/virtualmemoryscraper"
)

var standardMetrics = []string{
	"host/cpu/time",
	"host/memory/used",
	"host/disk/bytes",
	"host/disk/ops",
	"host/disk/time",
	"host/filesystem/used",
	"host/load/1m",
	"host/load/5m",
	"host/load/15m",
	"host/network/packets",
	"host/network/dropped_packets",
	"host/network/errors",
	"host/network/bytes",
	"host/network/tcp_connections",
	"host/swap/paging",
	"host/swap/usage",
}

var systemSpecificMetrics = map[string][]string{
	"linux":   {"host/filesystem/inodes/used", "host/swap/page_faults"},
	"darwin":  {"host/filesystem/inodes/used", "host/swap/page_faults"},
	"freebsd": {"host/filesystem/inodes/used", "host/swap/page_faults"},
	"openbsd": {"host/filesystem/inodes/used", "host/swap/page_faults"},
	"solaris": {"host/filesystem/inodes/used", "host/swap/page_faults"},
}

var factories = map[string]internal.Factory{
	cpuscraper.TypeStr:           &cpuscraper.Factory{},
	diskscraper.TypeStr:          &diskscraper.Factory{},
	filesystemscraper.TypeStr:    &filesystemscraper.Factory{},
	loadscraper.TypeStr:          &loadscraper.Factory{},
	memoryscraper.TypeStr:        &memoryscraper.Factory{},
	networkscraper.TypeStr:       &networkscraper.Factory{},
	virtualmemoryscraper.TypeStr: &virtualmemoryscraper.Factory{},
}

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		CollectionInterval: 100 * time.Millisecond,
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:           &cpuscraper.Config{ReportPerCPU: true},
			diskscraper.TypeStr:          &diskscraper.Config{},
			filesystemscraper.TypeStr:    &filesystemscraper.Config{},
			loadscraper.TypeStr:          &loadscraper.Config{},
			memoryscraper.TypeStr:        &memoryscraper.Config{},
			networkscraper.TypeStr:       &networkscraper.Config{},
			virtualmemoryscraper.TypeStr: &virtualmemoryscraper.Config{},
		},
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

func benchmarkScrapeMetrics(b *testing.B, cfg *Config) {
	sink := &exportertest.SinkMetricsExporter{}

	receiver, _ := newHostMetricsReceiver(context.Background(), zap.NewNop(), cfg, factories, sink)
	receiver.initializeScrapers(context.Background(), componenttest.NewNopHost())

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		receiver.scrapeMetrics(context.Background())
	}

	if len(sink.AllMetrics()) == 0 {
		b.Fail()
	}
}

func Benchmark_ScrapeCpuMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{cpuscraper.TypeStr: (&cpuscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeDiskMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{diskscraper.TypeStr: (&diskscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeFileSystemMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{filesystemscraper.TypeStr: (&filesystemscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeLoadMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{loadscraper.TypeStr: (&loadscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeMemoryMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{memoryscraper.TypeStr: (&memoryscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeNetworkMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{networkscraper.TypeStr: (&networkscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeVirtualMemoryMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{virtualmemoryscraper.TypeStr: (&virtualmemoryscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeDefaultMetrics(b *testing.B) {
	cfg := &Config{
		CollectionInterval: 100 * time.Millisecond,
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:           (&cpuscraper.Factory{}).CreateDefaultConfig(),
			diskscraper.TypeStr:          (&diskscraper.Factory{}).CreateDefaultConfig(),
			filesystemscraper.TypeStr:    (&filesystemscraper.Factory{}).CreateDefaultConfig(),
			loadscraper.TypeStr:          (&loadscraper.Factory{}).CreateDefaultConfig(),
			memoryscraper.TypeStr:        (&memoryscraper.Factory{}).CreateDefaultConfig(),
			networkscraper.TypeStr:       (&networkscraper.Factory{}).CreateDefaultConfig(),
			virtualmemoryscraper.TypeStr: (&virtualmemoryscraper.Factory{}).CreateDefaultConfig(),
		},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeAllMetrics(b *testing.B) {
	cfg := &Config{
		CollectionInterval: 100 * time.Millisecond,
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:           &cpuscraper.Config{ReportPerCPU: true},
			diskscraper.TypeStr:          &diskscraper.Config{},
			filesystemscraper.TypeStr:    &filesystemscraper.Config{},
			loadscraper.TypeStr:          &loadscraper.Config{},
			memoryscraper.TypeStr:        &memoryscraper.Config{},
			networkscraper.TypeStr:       &networkscraper.Config{},
			virtualmemoryscraper.TypeStr: &virtualmemoryscraper.Config{},
		},
	}

	benchmarkScrapeMetrics(b, cfg)
}
