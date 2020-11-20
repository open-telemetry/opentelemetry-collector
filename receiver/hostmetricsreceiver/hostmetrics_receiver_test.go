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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
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
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

var standardMetrics = []string{
	"system.cpu.time",
	"system.memory.usage",
	"system.disk.io",
	"system.disk.io_time",
	"system.disk.ops",
	"system.disk.operation_time",
	"system.disk.pending_operations",
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
	scraperFactories = factories
	resourceScraperFactories = resourceFactories

	sink := new(consumertest.MetricsSink)

	config := &Config{
		ScraperControllerSettings: scraperhelper.ScraperControllerSettings{
			CollectionInterval: 100 * time.Millisecond,
		},
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

	receiver, err := NewFactory().CreateMetricsReceiver(context.Background(), creationParams, config, sink)

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

		assertIncludesExpectedMetrics(t, got[0])
		return true
	}, waitFor, tick, "No metrics were collected after %v", waitFor)
}

func assertIncludesExpectedMetrics(t *testing.T, got pdata.Metrics) {
	// get the superset of metrics returned by all resource metrics (excluding the first)
	returnedMetrics := make(map[string]struct{})
	returnedResourceMetrics := make(map[string]struct{})
	rms := got.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		metrics := getMetricSlice(t, rm)
		returnedMetricNames := getReturnedMetricNames(metrics)
		if rm.Resource().Attributes().Len() == 0 {
			appendMapInto(returnedMetrics, returnedMetricNames)
		} else {
			appendMapInto(returnedResourceMetrics, returnedMetricNames)
		}
	}

	// verify the expected list of metrics returned (os dependent)
	expectedMetrics := append(standardMetrics, systemSpecificMetrics[runtime.GOOS]...)
	assert.Equal(t, len(expectedMetrics), len(returnedMetrics))
	for _, expected := range expectedMetrics {
		assert.Contains(t, returnedMetrics, expected)
	}

	// verify the expected list of resource metrics returned (Linux & Windows only)
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return
	}

	assert.Equal(t, len(resourceMetrics), len(returnedResourceMetrics))
	for _, expected := range resourceMetrics {
		assert.Contains(t, returnedResourceMetrics, expected)
	}
}

func getMetricSlice(t *testing.T, rm pdata.ResourceMetrics) pdata.MetricSlice {
	ilms := rm.InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilms.Len())
	return ilms.At(0).Metrics()
}

func getReturnedMetricNames(metrics pdata.MetricSlice) map[string]struct{} {
	metricNames := make(map[string]struct{})
	for i := 0; i < metrics.Len(); i++ {
		metricNames[metrics.At(i).Name()] = struct{}{}
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
func (m *mockFactory) CreateMetricsScraper(context.Context, *zap.Logger, internal.Config) (scraperhelper.MetricsScraper, error) {
	args := m.MethodCalled("CreateMetricsScraper")
	return args.Get(0).(scraperhelper.MetricsScraper), args.Error(1)
}

func (m *mockScraper) Name() string                                { return "" }
func (m *mockScraper) Start(context.Context, component.Host) error { return nil }
func (m *mockScraper) Shutdown(context.Context) error              { return nil }
func (m *mockScraper) Scrape(context.Context, string) (pdata.MetricSlice, error) {
	return pdata.NewMetricSlice(), errors.New("err1")
}

type mockResourceFactory struct{ mock.Mock }
type mockResourceScraper struct{ mock.Mock }

func (m *mockResourceFactory) CreateDefaultConfig() internal.Config { return &mockConfig{} }
func (m *mockResourceFactory) CreateResourceMetricsScraper(context.Context, *zap.Logger, internal.Config) (scraperhelper.ResourceMetricsScraper, error) {
	args := m.MethodCalled("CreateResourceMetricsScraper")
	return args.Get(0).(scraperhelper.ResourceMetricsScraper), args.Error(1)
}

func (m *mockResourceScraper) Name() string                                { return "" }
func (m *mockResourceScraper) Start(context.Context, component.Host) error { return nil }
func (m *mockResourceScraper) Shutdown(context.Context) error              { return nil }
func (m *mockResourceScraper) Scrape(context.Context, string) (pdata.ResourceMetricsSlice, error) {
	return pdata.NewResourceMetricsSlice(), errors.New("err2")
}

func TestGatherMetrics_ScraperKeyConfigError(t *testing.T) {
	scraperFactories = map[string]internal.ScraperFactory{}
	resourceScraperFactories = map[string]internal.ResourceScraperFactory{}

	sink := new(consumertest.MetricsSink)
	config := &Config{Scrapers: map[string]internal.Config{"error": &mockConfig{}}}
	_, err := NewFactory().CreateMetricsReceiver(context.Background(), creationParams, config, sink)
	require.Error(t, err)
}

func TestGatherMetrics_CreateMetricsScraperError(t *testing.T) {
	mFactory := &mockFactory{}
	mFactory.On("CreateMetricsScraper").Return(&mockScraper{}, errors.New("err1"))
	scraperFactories = map[string]internal.ScraperFactory{mockTypeStr: mFactory}
	resourceScraperFactories = map[string]internal.ResourceScraperFactory{}

	sink := new(consumertest.MetricsSink)
	config := &Config{Scrapers: map[string]internal.Config{mockTypeStr: &mockConfig{}}}
	_, err := NewFactory().CreateMetricsReceiver(context.Background(), creationParams, config, sink)
	require.Error(t, err)
}

func TestGatherMetrics_CreateMetricsResourceScraperError(t *testing.T) {
	mResourceFactory := &mockResourceFactory{}
	mResourceFactory.On("CreateResourceMetricsScraper").Return(&mockResourceScraper{}, errors.New("err1"))
	scraperFactories = map[string]internal.ScraperFactory{}
	resourceScraperFactories = map[string]internal.ResourceScraperFactory{mockResourceTypeStr: mResourceFactory}

	sink := new(consumertest.MetricsSink)
	config := &Config{Scrapers: map[string]internal.Config{mockResourceTypeStr: &mockConfig{}}}
	_, err := NewFactory().CreateMetricsReceiver(context.Background(), creationParams, config, sink)
	require.Error(t, err)
}

type notifyingSink struct {
	receivedMetrics bool
	timesCalled     int
	ch              chan int
}

func (s *notifyingSink) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	if md.MetricCount() > 0 {
		s.receivedMetrics = true
	}

	s.timesCalled++
	s.ch <- s.timesCalled
	return nil
}

func benchmarkScrapeMetrics(b *testing.B, cfg *Config) {
	scraperFactories = factories
	resourceScraperFactories = resourceFactories

	sink := &notifyingSink{ch: make(chan int, 10)}
	tickerCh := make(chan time.Time)

	options, err := createAddScraperOptions(context.Background(), zap.NewNop(), cfg, scraperFactories, resourceScraperFactories)
	require.NoError(b, err)
	options = append(options, scraperhelper.WithTickerChannel(tickerCh))

	receiver, err := scraperhelper.NewScraperControllerReceiver(&cfg.ScraperControllerSettings, zap.NewNop(), sink, options...)
	require.NoError(b, err)

	require.NoError(b, receiver.Start(context.Background(), componenttest.NewNopHost()))

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tickerCh <- time.Now()
		<-sink.ch
	}

	if !sink.receivedMetrics {
		b.Fail()
	}
}

func Benchmark_ScrapeCpuMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{cpuscraper.TypeStr: (&cpuscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeDiskMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{diskscraper.TypeStr: (&diskscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeFileSystemMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{filesystemscraper.TypeStr: (&filesystemscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeLoadMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{loadscraper.TypeStr: (&loadscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeMemoryMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{memoryscraper.TypeStr: (&memoryscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeNetworkMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{networkscraper.TypeStr: (&networkscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeProcessesMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{processesscraper.TypeStr: (&processesscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeSwapMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{swapscraper.TypeStr: (&swapscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeProcessMetrics(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		b.Skip("skipping test on non linux/windows")
	}

	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
		Scrapers:                  map[string]internal.Config{processscraper.TypeStr: (&processscraper.Factory{}).CreateDefaultConfig()},
	}

	benchmarkScrapeMetrics(b, cfg)
}

func Benchmark_ScrapeSystemMetrics(b *testing.B) {
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
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
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(""),
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
