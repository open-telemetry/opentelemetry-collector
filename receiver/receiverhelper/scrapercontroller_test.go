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

package receiverhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

type testInitialize struct {
	ch  chan bool
	err error
}

func (ts *testInitialize) initialize(context.Context) error {
	ts.ch <- true
	return ts.err
}

type testClose struct {
	ch  chan bool
	err error
}

func (ts *testClose) close(context.Context) error {
	ts.ch <- true
	return ts.err
}

type testScrapeMetrics struct {
	ch                chan int
	timesScrapeCalled int
	err               error
}

func (ts *testScrapeMetrics) scrape(_ context.Context) (pdata.MetricSlice, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled

	if ts.err != nil {
		return pdata.NewMetricSlice(), ts.err
	}

	return singleMetric(), nil
}

type testScrapeResourceMetrics struct {
	ch                chan int
	timesScrapeCalled int
	err               error
}

func (ts *testScrapeResourceMetrics) scrape(_ context.Context) (pdata.ResourceMetricsSlice, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled

	if ts.err != nil {
		return pdata.NewResourceMetricsSlice(), ts.err
	}

	return singleResourceMetric(), nil
}

type metricsTestCase struct {
	name string

	scrapers                  int
	resourceScrapers          int
	scraperControllerSettings *ScraperControllerSettings
	nilNextConsumer           bool
	scrapeErr                 error
	expectedNewErr            string
	expectScraped             bool

	initialize    bool
	close         bool
	initializeErr error
	closeErr      error
}

func TestScrapeController(t *testing.T) {
	testCases := []metricsTestCase{
		{
			name: "Standard",
		},
		{
			name:                      "AddMetricsScrapersWithCollectionInterval",
			scrapers:                  2,
			scraperControllerSettings: &ScraperControllerSettings{CollectionInterval: time.Millisecond},
			expectScraped:             true,
		},
		{
			name:            "AddMetricsScrapers_NilNextConsumerError",
			scrapers:        2,
			nilNextConsumer: true,
			expectedNewErr:  "nil nextConsumer",
		},
		{
			name:                      "AddMetricsScrapersWithCollectionInterval_InvalidCollectionIntervalError",
			scrapers:                  2,
			scraperControllerSettings: &ScraperControllerSettings{CollectionInterval: -time.Millisecond},
			expectedNewErr:            "collection_interval must be a positive duration",
		},
		{
			name:                      "AddMetricsScrapers_ScrapeError",
			scrapers:                  2,
			scraperControllerSettings: &ScraperControllerSettings{CollectionInterval: time.Millisecond},
			scrapeErr:                 errors.New("err1"),
		},
		{
			name:       "AddMetricsScrapersWithInitializeAndClose",
			scrapers:   2,
			initialize: true,
			close:      true,
		},
		{
			name:          "AddMetricsScrapersWithInitializeAndCloseErrors",
			scrapers:      2,
			initialize:    true,
			close:         true,
			initializeErr: errors.New("err1"),
			closeErr:      errors.New("err2"),
		},
		{
			name:                      "AddResourceMetricsScrapersWithCollectionInterval",
			resourceScrapers:          2,
			scraperControllerSettings: &ScraperControllerSettings{CollectionInterval: time.Millisecond},
			expectScraped:             true,
		},
		{
			name:             "AddResourceMetricsScrapers_NewError",
			resourceScrapers: 2,
			nilNextConsumer:  true,
			expectedNewErr:   "nil nextConsumer",
		},
		{
			name:                      "AddResourceMetricsScrapers_ScrapeError",
			resourceScrapers:          2,
			scraperControllerSettings: &ScraperControllerSettings{CollectionInterval: time.Millisecond},
			scrapeErr:                 errors.New("err1"),
		},
		{
			name:             "AddResourceMetricsScrapersWithInitializeAndClose",
			resourceScrapers: 2,
			initialize:       true,
			close:            true,
		},
		{
			name:             "AddResourceMetricsScrapersWithInitializeAndCloseErrors",
			resourceScrapers: 2,
			initialize:       true,
			close:            true,
			initializeErr:    errors.New("err1"),
			closeErr:         errors.New("err2"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			initializeChs := make([]chan bool, test.scrapers+test.resourceScrapers)
			scrapeMetricsChs := make([]chan int, test.scrapers)
			scrapeResourceMetricsChs := make([]chan int, test.resourceScrapers)
			closeChs := make([]chan bool, test.scrapers+test.resourceScrapers)
			options := configureMetricOptions(test, initializeChs, scrapeMetricsChs, scrapeResourceMetricsChs, closeChs)

			var nextConsumer consumer.MetricsConsumer
			sink := &exportertest.SinkMetricsExporter{}
			if !test.nilNextConsumer {
				nextConsumer = sink
			}
			defaultCfg := DefaultScraperControllerSettings("")
			cfg := &defaultCfg
			if test.scraperControllerSettings != nil {
				cfg = test.scraperControllerSettings
			}

			mr, err := NewScraperControllerReceiver(cfg, nextConsumer, options...)
			if test.expectedNewErr != "" {
				assert.EqualError(t, err, test.expectedNewErr)
				return
			}
			require.NoError(t, err)

			err = mr.Start(context.Background(), componenttest.NewNopHost())
			expectedStartErr := getExpectedStartErr(test)
			if expectedStartErr != nil {
				assert.Equal(t, expectedStartErr, err)
			} else if test.initialize {
				assertChannelsCalled(t, initializeChs, "initialize was not called")
			}

			// TODO: validate that observability information is reported correctly on error
			//       i.e. if test.scrapeErr != nil { ... }

			// validate that scrape is called at least 5 times for each configured scraper
			if test.expectScraped || test.scrapeErr != nil {
				for _, ch := range scrapeMetricsChs {
					require.Eventually(t, func() bool { return (<-ch) > 5 }, 500*time.Millisecond, time.Millisecond)
				}
				for _, ch := range scrapeResourceMetricsChs {
					require.Eventually(t, func() bool { return (<-ch) > 5 }, 500*time.Millisecond, time.Millisecond)
				}

				if test.expectScraped {
					assert.GreaterOrEqual(t, sink.MetricsCount(), 5)
				}
			}

			err = mr.Shutdown(context.Background())
			expectedShutdownErr := getExpectedShutdownErr(test)
			if expectedShutdownErr != nil {
				assert.EqualError(t, err, expectedShutdownErr.Error())
			} else if test.close {
				assertChannelsCalled(t, closeChs, "close was not called")
			}
		})
	}
}

func configureMetricOptions(test metricsTestCase, initializeChs []chan bool, scrapeMetricsChs, testScrapeResourceMetricsChs []chan int, closeChs []chan bool) []ScraperControllerOption {
	var metricOptions []ScraperControllerOption

	for i := 0; i < test.scrapers; i++ {
		var scraperOptions []ScraperOption
		if test.initialize {
			initializeChs[i] = make(chan bool, 1)
			ti := &testInitialize{ch: initializeChs[i], err: test.initializeErr}
			scraperOptions = append(scraperOptions, WithInitialize(ti.initialize))
		}
		if test.close {
			closeChs[i] = make(chan bool, 1)
			tc := &testClose{ch: closeChs[i], err: test.closeErr}
			scraperOptions = append(scraperOptions, WithClose(tc.close))
		}

		scrapeMetricsChs[i] = make(chan int, 10)
		tsm := &testScrapeMetrics{ch: scrapeMetricsChs[i], err: test.scrapeErr}
		metricOptions = append(metricOptions, AddMetricsScraper(NewMetricsScraper(tsm.scrape, scraperOptions...)))
	}

	for i := 0; i < test.resourceScrapers; i++ {
		var scraperOptions []ScraperOption
		if test.initialize {
			initializeChs[test.scrapers+i] = make(chan bool, 1)
			ti := &testInitialize{ch: initializeChs[test.scrapers+i], err: test.initializeErr}
			scraperOptions = append(scraperOptions, WithInitialize(ti.initialize))
		}
		if test.close {
			closeChs[test.scrapers+i] = make(chan bool, 1)
			tc := &testClose{ch: closeChs[test.scrapers+i], err: test.closeErr}
			scraperOptions = append(scraperOptions, WithClose(tc.close))
		}

		testScrapeResourceMetricsChs[i] = make(chan int, 10)
		tsrm := &testScrapeResourceMetrics{ch: testScrapeResourceMetricsChs[i], err: test.scrapeErr}
		metricOptions = append(metricOptions, AddResourceMetricsScraper(NewResourceMetricsScraper(tsrm.scrape, scraperOptions...)))
	}

	return metricOptions
}

func getExpectedStartErr(test metricsTestCase) error {
	return test.initializeErr
}

func getExpectedShutdownErr(test metricsTestCase) error {
	var errs []error

	if test.closeErr != nil {
		for i := 0; i < test.scrapers; i++ {
			errs = append(errs, test.closeErr)
		}
	}

	return componenterror.CombineErrors(errs)
}

func assertChannelsCalled(t *testing.T, chs []chan bool, message string) {
	for _, ic := range chs {
		assertChannelCalled(t, ic, message)
	}
}

func assertChannelCalled(t *testing.T, ch chan bool, message string) {
	select {
	case <-ch:
	default:
		assert.Fail(t, message)
	}
}

func singleMetric() pdata.MetricSlice {
	metrics := pdata.NewMetricSlice()
	metrics.Resize(1)
	return metrics
}

func singleResourceMetric() pdata.ResourceMetricsSlice {
	rms := pdata.NewResourceMetricsSlice()
	rms.Resize(1)
	rm := rms.At(0)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ilm := ilms.At(0)
	singleMetric().MoveAndAppendTo(ilm.Metrics())
	return rms
}

func TestSingleScrapePerTick(t *testing.T) {
	scrapeMetricsCh := make(chan int, 10)
	tsm := &testScrapeMetrics{ch: scrapeMetricsCh}

	scrapeResourceMetricsCh := make(chan int, 10)
	tsrm := &testScrapeResourceMetrics{ch: scrapeResourceMetricsCh}

	defaultCfg := DefaultScraperControllerSettings("")
	cfg := &defaultCfg

	tickerCh := make(chan time.Time)

	receiver, err := NewScraperControllerReceiver(
		cfg,
		&exportertest.SinkMetricsExporter{},
		AddMetricsScraper(NewMetricsScraper(tsm.scrape)),
		AddResourceMetricsScraper(NewResourceMetricsScraper(tsrm.scrape)),
		WithTickerChannel(tickerCh),
	)
	require.NoError(t, err)

	require.NoError(t, receiver.Start(context.Background(), componenttest.NewNopHost()))

	tickerCh <- time.Now()

	assert.Equal(t, 1, <-scrapeMetricsCh)
	assert.Equal(t, 1, <-scrapeResourceMetricsCh)

	select {
	case <-scrapeMetricsCh:
		assert.Fail(t, "Scrape was called more than once")
	case <-scrapeResourceMetricsCh:
		assert.Fail(t, "Scrape was called more than once")
	case <-time.After(100 * time.Millisecond):
		return
	}
}
