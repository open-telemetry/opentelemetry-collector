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

package hostmetricsreceiver

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/diskscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/filesystemscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/loadscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/networkscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/processesscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/processscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/swapscraper"
)

var standardMetrics = []string{
	"system.cpu.time",
	"system.memory.usage",
	"system.disk.io",
	"system.disk.ops",
	"system.disk.pending_operations",
	"system.disk.time",
	"system.filesystem.usage",
	"system.cpu.load_average.1m",
	"system.cpu.load_average.5m",
	"system.cpu.load_average.15m",
	"system.network.packets",
	"system.network.dropped_packets",
	"system.network.errors",
	"system.network.io",
	"system.network.tcp_connections",
	"system.swap.paging_ops",
	"system.swap.usage",
}

var resourceMetrics = []string{
	"process.cpu.time",
	"process.memory.physical_usage",
	"process.memory.virtual_usage",
	"process.disk.io",
}

var systemSpecificMetrics = map[string][]string{
	"linux":   {"system.disk.merged", "system.filesystem.inodes.usage", "system.processes.running", "system.processes.blocked", "system.swap.page_faults"},
	"darwin":  {"system.filesystem.inodes.usage", "system.processes.running", "system.processes.blocked", "system.swap.page_faults"},
	"freebsd": {"system.filesystem.inodes.usage", "system.processes.running", "system.processes.blocked", "system.swap.page_faults"},
	"openbsd": {"system.filesystem.inodes.usage", "system.processes.running", "system.processes.blocked", "system.swap.page_faults"},
	"solaris": {"system.filesystem.inodes.usage", "system.swap.page_faults"},
}

var factories = map[string]internal.ScraperFactory{
	cpuscraper.TypeStr:        &cpuscraper.Factory{},
	diskscraper.TypeStr:       &diskscraper.Factory{},
	filesystemscraper.TypeStr: &filesystemscraper.Factory{},
	loadscraper.TypeStr:       &loadscraper.Factory{},
	memoryscraper.TypeStr:     &memoryscraper.Factory{},
	networkscraper.TypeStr:    &networkscraper.Factory{},
	processesscraper.TypeStr:  &processesscraper.Factory{},
	swapscraper.TypeStr:       &swapscraper.Factory{},
}

var resourceFactories = map[string]internal.ResourceScraperFactory{
	processscraper.TypeStr: &processscraper.Factory{},
}

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		CollectionInterval: 100 * time.Millisecond,
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:        &cpuscraper.Config{},
			diskscraper.TypeStr:       &diskscraper.Config{},
			filesystemscraper.TypeStr: &filesystemscraper.Config{},
			loadscraper.TypeStr:       &loadscraper.Config{},
			memoryscraper.TypeStr:     &memoryscraper.Config{},
			networkscraper.TypeStr:    &networkscraper.Config{},
			processesscraper.TypeStr:  &processesscraper.Config{},
			swapscraper.TypeStr:       &swapscraper.Config{},
		},
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		config.Scrapers[processscraper.TypeStr] = &processscraper.Config{}
	}

	receiver, err := newHostMetricsReceiver(context.Background(), zap.NewNop(), config, factories, resourceFactories, sink)

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	ctx, cancelFn := context.WithCancel(context.Background())
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	// canceling the context provided to Start should not cancel any async processes initiated by the receiver
	cancelFn()

	const tick = 50 * time.Millisecond
	const waitFor = 5 * time.Second
	require.Eventuallyf(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertIncludesStandardMetrics(t, got[0])
		assertIncludesResourceMetrics(t, got[0])
		return true
	}, waitFor, tick, "No metrics were collected after %v", waitFor)
}

func assertIncludesStandardMetrics(t *testing.T, got pdata.Metrics) {
	md := pdatautil.MetricsToInternalMetrics(got)

	// get the first ResourceMetrics object
	rms := md.ResourceMetrics()
	require.GreaterOrEqual(t, rms.Len(), 1)
	rm := rms.At(0)
	assert.True(t, rm.Resource().IsNil() || rm.Resource().Attributes().Len() == 0)

	metrics := getMetricSlice(t, rm)
	returnedMetrics := getReturnedMetricNames(metrics)

	// the expected list of metrics returned is os dependent
	expectedMetrics := append(standardMetrics, systemSpecificMetrics[runtime.GOOS]...)
	assert.Equal(t, len(expectedMetrics), len(returnedMetrics))
	for _, expected := range expectedMetrics {
		assert.Contains(t, returnedMetrics, expected)
	}
}

func assertIncludesResourceMetrics(t *testing.T, got pdata.Metrics) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return
	}

	md := pdatautil.MetricsToInternalMetrics(got)

	// get the superset of metrics returned by all resource metrics (excluding the first)
	returnedMetrics := make(map[string]struct{})
	rms := md.ResourceMetrics()
	for i := 1; i < rms.Len(); i++ {
		rm := rms.At(i)
		assert.Greater(t, rm.Resource().Attributes().Len(), 0)
		metrics := getMetricSlice(t, rm)
		appendMapInto(returnedMetrics, getReturnedMetricNames(metrics))
	}

	assert.Equal(t, len(resourceMetrics), len(returnedMetrics))
	for _, expected := range resourceMetrics {
		assert.Contains(t, returnedMetrics, expected)
	}
}

func getMetricSlice(t *testing.T, rm dataold.ResourceMetrics) dataold.MetricSlice {
	ilms := rm.InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilms.Len())
	return ilms.At(0).Metrics()
}

func getReturnedMetricNames(metrics dataold.MetricSlice) map[string]struct{} {
	metricNames := make(map[string]struct{})
	for i := 0; i < metrics.Len(); i++ {
		metricNames[metrics.At(i).MetricDescriptor().Name()] = struct{}{}
	}
	return metricNames
}

func appendMapInto(m1 map[string]struct{}, m2 map[string]struct{}) {
	for k, v := range m2 {
		m1[k] = v
	}
}

const mockTypeStr = "mock"
const mockResourceTypeStr = "mockresource"

type mockConfig struct{}

type mockFactory struct{ mock.Mock }
type mockScraper struct{ mock.Mock }

func (m *mockFactory) CreateDefaultConfig() internal.Config { return &mockConfig{} }
func (m *mockFactory) CreateMetricsScraper(ctx context.Context, logger *zap.Logger, cfg internal.Config) (internal.Scraper, error) {
	args := m.MethodCalled("CreateMetricsScraper")
	return args.Get(0).(internal.Scraper), args.Error(1)
}

func (m *mockScraper) Initialize(ctx context.Context) error { return nil }
func (m *mockScraper) Close(ctx context.Context) error      { return nil }
func (m *mockScraper) ScrapeMetrics(ctx context.Context) (dataold.MetricSlice, error) {
	return dataold.NewMetricSlice(), errors.New("err1")
}

type mockResourceFactory struct{ mock.Mock }
type mockResourceScraper struct{ mock.Mock }

func (m *mockResourceFactory) CreateDefaultConfig() internal.Config { return &mockConfig{} }
func (m *mockResourceFactory) CreateMetricsScraper(ctx context.Context, logger *zap.Logger, cfg internal.Config) (internal.ResourceScraper, error) {
	args := m.MethodCalled("CreateMetricsScraper")
	return args.Get(0).(internal.ResourceScraper), args.Error(1)
}

func (m *mockResourceScraper) Initialize(ctx context.Context) error { return nil }
func (m *mockResourceScraper) Close(ctx context.Context) error      { return nil }
func (m *mockResourceScraper) ScrapeMetrics(ctx context.Context) (dataold.ResourceMetricsSlice, error) {
	return dataold.NewResourceMetricsSlice(), errors.New("err2")
}

func TestGatherMetrics_ScraperKeyConfigError(t *testing.T) {
	var mockFactories = map[string]internal.ScraperFactory{}
	var mockResourceFactories = map[string]internal.ResourceScraperFactory{}

	sink := &exportertest.SinkMetricsExporter{}
	config := &Config{Scrapers: map[string]internal.Config{"error": &mockConfig{}}}

	_, err := newHostMetricsReceiver(context.Background(), zap.NewNop(), config, mockFactories, mockResourceFactories, sink)
	require.Error(t, err)
}

func TestGatherMetrics_CreateMetricsScraperError(t *testing.T) {
	mFactory := &mockFactory{}
	mFactory.On("CreateMetricsScraper").Return(&mockScraper{}, errors.New("err1"))
	var mockFactories = map[string]internal.ScraperFactory{mockTypeStr: mFactory}
	var mockResourceFactories = map[string]internal.ResourceScraperFactory{}

	sink := &exportertest.SinkMetricsExporter{}
	config := &Config{Scrapers: map[string]internal.Config{mockTypeStr: &mockConfig{}}}
	_, err := newHostMetricsReceiver(context.Background(), zap.NewNop(), config, mockFactories, mockResourceFactories, sink)
	require.Error(t, err)
}

func TestGatherMetrics_CreateMetricsResourceScraperError(t *testing.T) {
	mResourceFactory := &mockResourceFactory{}
	mResourceFactory.On("CreateMetricsScraper").Return(&mockResourceScraper{}, errors.New("err1"))
	var mockFactories = map[string]internal.ScraperFactory{}
	var mockResourceFactories = map[string]internal.ResourceScraperFactory{mockTypeStr: mResourceFactory}

	sink := &exportertest.SinkMetricsExporter{}
	config := &Config{Scrapers: map[string]internal.Config{mockTypeStr: &mockConfig{}}}
	_, err := newHostMetricsReceiver(context.Background(), zap.NewNop(), config, mockFactories, mockResourceFactories, sink)
	require.Error(t, err)
}

func TestGatherMetrics_Error(t *testing.T) {
	mFactory := &mockFactory{}
	mFactory.On("CreateMetricsScraper").Return(&mockScraper{}, nil)
	mResourceFactory := &mockResourceFactory{}
	mResourceFactory.On("CreateMetricsScraper").Return(&mockResourceScraper{}, nil)

	var mockFactories = map[string]internal.ScraperFactory{mockTypeStr: mFactory}
	var mockResourceFactories = map[string]internal.ResourceScraperFactory{mockResourceTypeStr: mResourceFactory}

	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		Scrapers: map[string]internal.Config{
			mockTypeStr:         &mockConfig{},
			mockResourceTypeStr: &mockConfig{},
		},
	}

	receiver, err := newHostMetricsReceiver(context.Background(), zap.NewNop(), config, mockFactories, mockResourceFactories, sink)
	require.NoError(t, err)

	receiver.initializeScrapers(context.Background(), componenttest.NewNopHost())
	receiver.scrapeMetrics(context.Background())

	got := sink.AllMetrics()

	// expect to get one empty resource metrics entry
	require.Equal(t, 1, len(got))
	rm := pdatautil.MetricsToInternalMetrics(got[0]).ResourceMetrics()
	require.Equal(t, 1, rm.Len())
	ilm := rm.At(0).InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilm.Len())
	metrics := ilm.At(0).Metrics()
	require.Equal(t, 0, metrics.Len())
}

func benchmarkScrapeMetrics(b *testing.B, cfg *Config) {
	sink := &exportertest.SinkMetricsExporter{}

	receiver, _ := newHostMetricsReceiver(context.Background(), zap.NewNop(), cfg, factories, resourceFactories, sink)
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

func Benchmark_ScrapeProcessesMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{processesscraper.TypeStr: (&processesscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeSwapMetrics(b *testing.B) {
	cfg := &Config{Scrapers: map[string]internal.Config{swapscraper.TypeStr: (&swapscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeProcessMetrics(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		b.Skip("skipping test on non linux/windows")
	}

	cfg := &Config{Scrapers: map[string]internal.Config{processscraper.TypeStr: (&processscraper.Factory{}).CreateDefaultConfig()}}
	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeSystemMetrics(b *testing.B) {
	cfg := &Config{
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:        (&cpuscraper.Factory{}).CreateDefaultConfig(),
			diskscraper.TypeStr:       (&diskscraper.Factory{}).CreateDefaultConfig(),
			filesystemscraper.TypeStr: (&filesystemscraper.Factory{}).CreateDefaultConfig(),
			loadscraper.TypeStr:       (&loadscraper.Factory{}).CreateDefaultConfig(),
			memoryscraper.TypeStr:     (&memoryscraper.Factory{}).CreateDefaultConfig(),
			networkscraper.TypeStr:    (&networkscraper.Factory{}).CreateDefaultConfig(),
			processesscraper.TypeStr:  (&processesscraper.Factory{}).CreateDefaultConfig(),
			swapscraper.TypeStr:       (&swapscraper.Factory{}).CreateDefaultConfig(),
		},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeSystemAndProcessMetrics(b *testing.B) {
	cfg := &Config{
		Scrapers: map[string]internal.Config{
			cpuscraper.TypeStr:        &cpuscraper.Config{},
			diskscraper.TypeStr:       &diskscraper.Config{},
			filesystemscraper.TypeStr: &filesystemscraper.Config{},
			loadscraper.TypeStr:       &loadscraper.Config{},
			memoryscraper.TypeStr:     &memoryscraper.Config{},
			networkscraper.TypeStr:    &networkscraper.Config{},
			processesscraper.TypeStr:  &processesscraper.Config{},
			swapscraper.TypeStr:       &swapscraper.Config{},
		},
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		cfg.Scrapers[processscraper.TypeStr] = &processscraper.Config{}
	}

	benchmarkScrapeMetrics(b, cfg)
}
