// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
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

func (ts *testScrapeMetrics) scrape(context.Context) (pmetric.Metrics, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled

	if ts.err != nil {
		return pmetric.Metrics{}, ts.err
	}

	md := pmetric.NewMetrics()
	md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()
	return md, nil
}

func newTestNoDelaySettings() *ControllerConfig {
	return &ControllerConfig{
		CollectionInterval: time.Second,
		InitialDelay:       0,
	}
}

type metricsTestCase struct {
	name string

	scrapers                  int
	scraperControllerSettings *ControllerConfig
	scrapeErr                 error
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
			name:      "AddMetricsScrapers_ScrapeError",
			scrapers:  2,
			scrapeErr: errors.New("err1"),
		},
		{
			name:          "AddMetricsScrapersWithInitializeAndClose",
			scrapers:      2,
			initialize:    true,
			expectScraped: true,
			close:         true,
		},
		{
			name:          "AddMetricsScrapersWithInitializeAndCloseErrors",
			scrapers:      2,
			initialize:    true,
			close:         true,
			initializeErr: errors.New("err1"),
			closeErr:      errors.New("err2"),
		},
	}

	for _, tt := range testCases {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			receiverID := component.MustNewID("receiver")
			tt, err := componenttest.SetupTelemetry(receiverID)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			initializeChs := make([]chan bool, test.scrapers)
			scrapeMetricsChs := make([]chan int, test.scrapers)
			closeChs := make([]chan bool, test.scrapers)
			options := configureMetricOptions(t, test, initializeChs, scrapeMetricsChs, closeChs)

			tickerCh := make(chan time.Time)
			options = append(options, WithTickerChannel(tickerCh))

			sink := new(consumertest.MetricsSink)
			cfg := newTestNoDelaySettings()
			if test.scraperControllerSettings != nil {
				cfg = test.scraperControllerSettings
			}

			mr, err := NewScraperControllerReceiver(cfg, receiver.Settings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, sink, options...)
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
				for _, ch := range scrapeMetricsChs {
					<-ch
				}
				// Consume the initial scrapes on start
				for i := 0; i < iterations; i++ {
					tickerCh <- time.Now()

					for _, ch := range scrapeMetricsChs {
						<-ch
					}
				}

				// wait until all calls to scrape have completed
				if test.scrapeErr == nil {
					require.Eventually(t, func() bool {
						return sink.DataPointCount() == (1+iterations)*(test.scrapers)
					}, time.Second, time.Millisecond)
				}

				if test.expectScraped {
					assert.GreaterOrEqual(t, sink.DataPointCount(), iterations)
				}

				spans := tt.SpanRecorder.Ended()
				assertReceiverSpan(t, spans)
				assertReceiverViews(t, tt, sink)
				assertScraperSpan(t, test.scrapeErr, spans)
				assertScraperViews(t, tt, test.scrapeErr, sink)
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

func configureMetricOptions(t *testing.T, test metricsTestCase, initializeChs []chan bool, scrapeMetricsChs []chan int, closeChs []chan bool) []ScraperControllerOption {
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
		scp, err := NewScraper(tsm.scrape, scraperOptions...)
		require.NoError(t, err)

		metricOptions = append(metricOptions, AddScraper(component.MustNewType("scraper"), scp))
	}

	return metricOptions
}

func getExpectedStartErr(test metricsTestCase) error {
	return test.initializeErr
}

func getExpectedShutdownErr(test metricsTestCase) error {
	var errs error

	if test.closeErr != nil {
		for i := 0; i < test.scrapers; i++ {
			errs = multierr.Append(errs, test.closeErr)
		}
	}

	return errs
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

func assertReceiverViews(t *testing.T, tt componenttest.TestTelemetry, sink *consumertest.MetricsSink) {
	dataPointCount := 0
	for _, md := range sink.AllMetrics() {
		dataPointCount += md.DataPointCount()
	}
	require.NoError(t, tt.CheckReceiverMetrics("", int64(dataPointCount), 0))
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
		if span.Name() == "scraper/scraper/ScrapeMetrics" {
			scraperSpan = true
			assert.Equal(t, expectedStatusCode, span.Status().Code)
			assert.Equal(t, expectedStatusMessage, span.Status().Description)
			break
		}
	}
	assert.True(t, scraperSpan)
}

func assertScraperViews(t *testing.T, tt componenttest.TestTelemetry, expectedErr error, sink *consumertest.MetricsSink) {
	expectedScraped := int64(sink.DataPointCount())
	expectedErrored := int64(0)
	if expectedErr != nil {
		var partialError scrapererror.PartialScrapeError
		if errors.As(expectedErr, &partialError) {
			expectedErrored = int64(partialError.Failed)
		} else {
			expectedScraped = int64(0)
			expectedErrored = int64(sink.DataPointCount())
		}
	}

	require.NoError(t, tt.CheckScraperMetrics(component.MustNewID("receiver"), component.MustNewID("scraper"), expectedScraped, expectedErrored))
}

func TestSingleScrapePerInterval(t *testing.T) {
	scrapeMetricsCh := make(chan int, 10)
	tsm := &testScrapeMetrics{ch: scrapeMetricsCh}

	cfg := newTestNoDelaySettings()

	tickerCh := make(chan time.Time)

	scp, err := NewScraper(tsm.scrape)
	require.NoError(t, err)

	recv, err := NewScraperControllerReceiver(
		cfg,
		receivertest.NewNopSettings(),
		new(consumertest.MetricsSink),
		AddScraper(component.MustNewType("scaper"), scp),
		WithTickerChannel(tickerCh),
	)
	require.NoError(t, err)

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, recv.Shutdown(context.Background())) }()

	tickerCh <- time.Now()

	assert.Eventually(
		t,
		func() bool {
			return <-scrapeMetricsCh == 2
		},
		300*time.Millisecond,
		100*time.Millisecond,
		"Make sure the scraper channel is called twice",
	)

	select {
	case <-scrapeMetricsCh:
		assert.Fail(t, "Scrape was called more than twice")
	case <-time.After(100 * time.Millisecond):
		return
	}
}

func TestScrapeControllerStartsOnInit(t *testing.T) {
	t.Parallel()

	tsm := &testScrapeMetrics{
		ch: make(chan int, 1),
	}

	scp, err := NewScraper(tsm.scrape)
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewScraperControllerReceiver(
		&ControllerConfig{
			CollectionInterval: time.Hour,
			InitialDelay:       0,
		},
		receivertest.NewNopSettings(),
		new(consumertest.MetricsSink),
		AddScraper(component.MustNewType("scaper"), scp),
	)
	require.NoError(t, err, "Must not error when creating scrape controller")

	assert.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()), "Must not error on start")
	<-time.After(500 * time.Nanosecond)
	require.NoError(t, r.Shutdown(context.Background()), "Must not have errored on shutdown")
	assert.Equal(t, 1, tsm.timesScrapeCalled, "Must have been called as soon as the controller started")
}

func TestScrapeControllerInitialDelay(t *testing.T) {
	if testing.Short() {
		t.Skip("This requires real time to pass, skipping")
		return
	}

	t.Parallel()

	var (
		elapsed = make(chan time.Time, 1)
		cfg     = ControllerConfig{
			CollectionInterval: time.Second,
			InitialDelay:       300 * time.Millisecond,
		}
	)

	scp, err := NewScraper(func(context.Context) (pmetric.Metrics, error) {
		elapsed <- time.Now()
		return pmetric.NewMetrics(), nil
	})
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewScraperControllerReceiver(
		&cfg,
		receivertest.NewNopSettings(),
		new(consumertest.MetricsSink),
		AddScraper(component.MustNewType("scaper"), scp),
	)
	require.NoError(t, err, "Must not error when creating receiver")

	t0 := time.Now()
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()), "Must not error when starting")
	t1 := <-elapsed

	assert.GreaterOrEqual(t, t1.Sub(t0), 300*time.Millisecond, "Must have had 300ms pass as defined by initial delay")

	assert.NoError(t, r.Shutdown(context.Background()), "Must not error closing down")
}

func TestShutdownBeforeScrapeCanStart(t *testing.T) {
	cfg := ControllerConfig{
		CollectionInterval: time.Second,
		InitialDelay:       5 * time.Second,
	}

	scp, err := NewScraper(func(context.Context) (pmetric.Metrics, error) {
		// make the scraper wait for long enough it would disrupt a shutdown.
		time.Sleep(30 * time.Second)
		return pmetric.NewMetrics(), nil
	})
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewScraperControllerReceiver(
		&cfg,
		receivertest.NewNopSettings(),
		new(consumertest.MetricsSink),
		AddScraper(component.MustNewType("scaper"), scp),
	)
	require.NoError(t, err, "Must not error when creating receiver")
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	shutdown := make(chan struct{}, 1)
	go func() {
		assert.NoError(t, r.Shutdown(context.Background()))
		close(shutdown)
	}()
	timer := time.NewTicker(10 * time.Second)
	select {
	case <-timer.C:
		require.Fail(t, "shutdown should not wait for scraping")
	case <-shutdown:
	}
}
