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

package scraperhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

type testInitialize struct {
	ch  chan bool
	err error
}

func (ts *testInitialize) start(context.Context, component.Host) error {
	ts.ch <- true
	return ts.err
}

type testClose struct {
	ch  chan bool
	err error
}

func (ts *testClose) shutdown(context.Context) error {
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
		return pdata.MetricSlice{}, ts.err
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
		return pdata.ResourceMetricsSlice{}, ts.err
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
			name: "NoScrapers",
		},
		{
			name:          "AddMetricsScrapersWithCollectionInterval",
			scrapers:      2,
			expectScraped: true,
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
			name:      "AddMetricsScrapers_ScrapeError",
			scrapers:  2,
			scrapeErr: errors.New("err1"),
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
			name:             "AddResourceMetricsScrapersWithCollectionInterval",
			resourceScrapers: 2,
			expectScraped:    true,
		},
		{
			name:             "AddResourceMetricsScrapers_NewError",
			resourceScrapers: 2,
			nilNextConsumer:  true,
			expectedNewErr:   "nil nextConsumer",
		},
		{
			name:             "AddResourceMetricsScrapers_ScrapeError",
			resourceScrapers: 2,
			scrapeErr:        errors.New("err1"),
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
			sr := new(tracetest.SpanRecorder)
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			otel.SetTracerProvider(tp)
			defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

			done, err := obsreporttest.SetupRecordedMetricsTest()
			require.NoError(t, err)
			defer done()

			initializeChs := make([]chan bool, test.scrapers+test.resourceScrapers)
			scrapeMetricsChs := make([]chan int, test.scrapers)
			scrapeResourceMetricsChs := make([]chan int, test.resourceScrapers)
			closeChs := make([]chan bool, test.scrapers+test.resourceScrapers)
			options := configureMetricOptions(test, initializeChs, scrapeMetricsChs, scrapeResourceMetricsChs, closeChs)

			tickerCh := make(chan time.Time)
			options = append(options, WithTickerChannel(tickerCh))

			var nextConsumer consumer.Metrics
			sink := new(consumertest.MetricsSink)
			if !test.nilNextConsumer {
				nextConsumer = sink
			}
			defaultCfg := DefaultScraperControllerSettings("receiver")
			cfg := &defaultCfg
			if test.scraperControllerSettings != nil {
				cfg = test.scraperControllerSettings
			}

			mr, err := NewScraperControllerReceiver(cfg, zap.NewNop(), nextConsumer, options...)
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
				assertChannelsCalled(t, initializeChs, "start was not called")
			}

			const iterations = 5

			if test.expectScraped || test.scrapeErr != nil {
				// validate that scrape is called at least N times for each configured scraper
				for i := 0; i < iterations; i++ {
					tickerCh <- time.Now()

					for _, ch := range scrapeMetricsChs {
						<-ch
					}
					for _, ch := range scrapeResourceMetricsChs {
						<-ch
					}
				}

				// wait until all calls to scrape have completed
				if test.scrapeErr == nil {
					require.Eventually(t, func() bool {
						return sink.DataPointCount() == iterations*(test.scrapers+test.resourceScrapers)
					}, time.Second, time.Millisecond)
				}

				if test.expectScraped {
					assert.GreaterOrEqual(t, sink.DataPointCount(), iterations)
				}

				spans := sr.Ended()
				assertReceiverSpan(t, spans)
				assertReceiverViews(t, sink)
				assertScraperSpan(t, test.scrapeErr, spans)
				assertScraperViews(t, test.scrapeErr, sink)
			}

			err = mr.Shutdown(context.Background())
			expectedShutdownErr := getExpectedShutdownErr(test)
			if expectedShutdownErr != nil {
				assert.EqualError(t, err, expectedShutdownErr.Error())
			} else if test.close {
				assertChannelsCalled(t, closeChs, "shutdown was not called")
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
			scraperOptions = append(scraperOptions, WithStart(ti.start))
		}
		if test.close {
			closeChs[i] = make(chan bool, 1)
			tc := &testClose{ch: closeChs[i], err: test.closeErr}
			scraperOptions = append(scraperOptions, WithShutdown(tc.shutdown))
		}

		scrapeMetricsChs[i] = make(chan int)
		tsm := &testScrapeMetrics{ch: scrapeMetricsChs[i], err: test.scrapeErr}
		metricOptions = append(metricOptions, AddScraper(NewMetricsScraper("scraper", tsm.scrape, scraperOptions...)))
	}

	for i := 0; i < test.resourceScrapers; i++ {
		var scraperOptions []ScraperOption
		if test.initialize {
			initializeChs[test.scrapers+i] = make(chan bool, 1)
			ti := &testInitialize{ch: initializeChs[test.scrapers+i], err: test.initializeErr}
			scraperOptions = append(scraperOptions, WithStart(ti.start))
		}
		if test.close {
			closeChs[test.scrapers+i] = make(chan bool, 1)
			tc := &testClose{ch: closeChs[test.scrapers+i], err: test.closeErr}
			scraperOptions = append(scraperOptions, WithShutdown(tc.shutdown))
		}

		testScrapeResourceMetricsChs[i] = make(chan int)
		tsrm := &testScrapeResourceMetrics{ch: testScrapeResourceMetricsChs[i], err: test.scrapeErr}
		metricOptions = append(metricOptions, AddScraper(NewResourceMetricsScraper(config.NewID("scraper"), tsrm.scrape, scraperOptions...)))
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

	return consumererror.Combine(errs)
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

func assertReceiverSpan(t *testing.T, spans []sdktrace.ReadOnlySpan) {
	receiverSpan := false
	for _, span := range spans {
		if span.Name() == "receiver/receiver/MetricsReceived" {
			receiverSpan = true
			break
		}
	}
	assert.True(t, receiverSpan)
}

func assertReceiverViews(t *testing.T, sink *consumertest.MetricsSink) {
	dataPointCount := 0
	for _, md := range sink.AllMetrics() {
		dataPointCount += md.DataPointCount()
	}
	obsreporttest.CheckReceiverMetrics(t, config.NewID("receiver"), "", int64(dataPointCount), 0)
}

func assertScraperSpan(t *testing.T, expectedErr error, spans []sdktrace.ReadOnlySpan) {
	expectedStatusCode := codes.Unset
	expectedStatusMessage := ""
	if expectedErr != nil {
		expectedStatusCode = codes.Error
		expectedStatusMessage = expectedErr.Error()
	}

	scraperSpan := false
	for _, span := range spans {
		if span.Name() == "scraper/receiver/scraper/MetricsScraped" {
			scraperSpan = true
			assert.Equal(t, expectedStatusCode, span.Status().Code)
			assert.Equal(t, expectedStatusMessage, span.Status().Description)
			break
		}
	}
	assert.True(t, scraperSpan)
}

func assertScraperViews(t *testing.T, expectedErr error, sink *consumertest.MetricsSink) {
	expectedScraped := int64(sink.DataPointCount())
	expectedErrored := int64(0)
	if expectedErr != nil {
		if partialError, isPartial := expectedErr.(scrapererror.PartialScrapeError); isPartial {
			expectedErrored = int64(partialError.Failed)
		} else {
			expectedScraped = int64(0)
			expectedErrored = int64(sink.DataPointCount())
		}
	}

	obsreporttest.CheckScraperMetrics(t, config.NewID("receiver"), config.NewID("scraper"), expectedScraped, expectedErrored)
}

func singleMetric() pdata.MetricSlice {
	metrics := pdata.NewMetricSlice()
	metric := metrics.AppendEmpty()
	metric.SetDataType(pdata.MetricDataTypeGauge)
	metric.Gauge().DataPoints().AppendEmpty()
	return metrics
}

func singleResourceMetric() pdata.ResourceMetricsSlice {
	rms := pdata.NewResourceMetricsSlice()
	ilm := rms.AppendEmpty().InstrumentationLibraryMetrics().AppendEmpty()
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
		zap.NewNop(),
		new(consumertest.MetricsSink),
		AddScraper(NewMetricsScraper("", tsm.scrape)),
		AddScraper(NewResourceMetricsScraper(config.NewID("scraper"), tsrm.scrape)),
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
