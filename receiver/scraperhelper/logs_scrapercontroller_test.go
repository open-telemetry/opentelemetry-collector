// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
)

//type testInitialize struct {
//	ch  chan bool
//	err error
//}

//func (ts *testInitialize) start(context.Context, component.Host) error {
//	ts.ch <- true
//	return ts.err
//}
//
//type testClose struct {
//	ch  chan bool
//	err error
//}

//func (ts *testClose) shutdown(context.Context) error {
//	ts.ch <- true
//	return ts.err
//}

type testScrapeLogs struct {
	ch                chan int
	timesScrapeCalled int
	err               error
}

func (ts *testScrapeLogs) scrape(context.Context) (plog.Logs, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled

	if ts.err != nil {
		return plog.Logs{}, ts.err
	}

	md := plog.NewLogs()
	md.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("test")
	return md, nil
}

//func newTestNoDelaySettings() *ControllerConfig {
//	return &ControllerConfig{
//		CollectionInterval: time.Second,
//		InitialDelay:       0,
//	}
//}

type logsTestCase struct {
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

func TestLogsScrapeController(t *testing.T) {
	testCases := []logsTestCase{
		{
			name: "NoScrapers",
		},
		{
			name:          "AddLogsScrapersWithCollectionInterval",
			scrapers:      2,
			expectScraped: true,
		},
		{
			name:      "AddLogsScrapers_ScrapeError",
			scrapers:  2,
			scrapeErr: errors.New("err1"),
		},
		{
			name:          "AddLogsScrapersWithInitializeAndClose",
			scrapers:      2,
			initialize:    true,
			expectScraped: true,
			close:         true,
		},
		{
			name:          "AddLogsScrapersWithInitializeAndCloseErrors",
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
			scrapeLogsChs := make([]chan int, test.scrapers)
			closeChs := make([]chan bool, test.scrapers)
			options := configureLogOptions(t, test, initializeChs, scrapeLogsChs, closeChs)

			tickerCh := make(chan time.Time)
			options = append(options, WithLogsTickerChannel(tickerCh))

			sink := new(consumertest.LogsSink)
			cfg := newTestNoDelaySettings()
			if test.scraperControllerSettings != nil {
				cfg = test.scraperControllerSettings
			}

			mr, err := NewLogsScraperControllerReceiver(cfg, receiver.Settings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, sink, options...)
			require.NoError(t, err)

			err = mr.Start(context.Background(), componenttest.NewNopHost())
			expectedStartErr := getLogsExpectedStartErr(test)
			if expectedStartErr != nil {
				assert.Equal(t, expectedStartErr, err)
			} else if test.initialize {
				assertChannelsCalled(t, initializeChs, "start was not called")
			}

			const iterations = 5

			if test.expectScraped || test.scrapeErr != nil {
				// validate that scrape is called at least N times for each configured scraper
				for _, ch := range scrapeLogsChs {
					<-ch
				}
				// Consume the initial scrapes on start
				for i := 0; i < iterations; i++ {
					tickerCh <- time.Now()

					for _, ch := range scrapeLogsChs {
						<-ch
					}
				}

				// wait until all calls to scrape have completed
				if test.scrapeErr == nil {
					require.Eventually(t, func() bool {
						return sink.LogRecordCount() == (1+iterations)*(test.scrapers)
					}, time.Second, time.Millisecond)
				}

				if test.expectScraped {
					assert.GreaterOrEqual(t, sink.LogRecordCount(), iterations)
				}

				spans := tt.SpanRecorder.Ended()
				assertLogsReceiverSpan(t, spans)
				assertLogsReceiverViews(t, tt, sink)
				assertLogsScraperSpan(t, test.scrapeErr, spans)
				assertLogsScraperViews(t, tt, test.scrapeErr, sink)
			}

			err = mr.Shutdown(context.Background())
			expectedShutdownErr := getLogsExpectedShutdownErr(test)
			if expectedShutdownErr != nil {
				assert.EqualError(t, err, expectedShutdownErr.Error())
			} else if test.close {
				assertChannelsCalled(t, closeChs, "shutdown was not called")
			}
		})
	}
}

func configureLogOptions(t *testing.T, test logsTestCase, initializeChs []chan bool, scrapeLogsChs []chan int, closeChs []chan bool) []LogsScraperControllerOption {
	var logOptions []LogsScraperControllerOption

	for i := 0; i < test.scrapers; i++ {
		var scraperOptions []scraper.Option
		if test.initialize {
			initializeChs[i] = make(chan bool, 1)
			ti := &testInitialize{ch: initializeChs[i], err: test.initializeErr}
			scraperOptions = append(scraperOptions, scraper.WithStart(ti.start))
		}
		if test.close {
			closeChs[i] = make(chan bool, 1)
			tc := &testClose{ch: closeChs[i], err: test.closeErr}
			scraperOptions = append(scraperOptions, scraper.WithShutdown(tc.shutdown))
		}

		scrapeLogsChs[i] = make(chan int)
		tsm := &testScrapeLogs{ch: scrapeLogsChs[i], err: test.scrapeErr}
		scp, err := scraper.NewLogs(tsm.scrape, scraperOptions...)
		require.NoError(t, err)

		logOptions = append(logOptions, AddLogsScraper(component.MustNewType("scraper"), scp))
	}

	return logOptions
}

func getLogsExpectedStartErr(test logsTestCase) error {
	return test.initializeErr
}

func getLogsExpectedShutdownErr(test logsTestCase) error {
	var errs error

	if test.closeErr != nil {
		for i := 0; i < test.scrapers; i++ {
			errs = multierr.Append(errs, test.closeErr)
		}
	}

	return errs
}

func assertLogsChannelsCalled(t *testing.T, chs []chan bool, message string) {
	for _, ic := range chs {
		assertChannelCalled(t, ic, message)
	}
}

func assertLogsChannelCalled(t *testing.T, ch chan bool, message string) {
	select {
	case <-ch:
	default:
		assert.Fail(t, message)
	}
}

func assertLogsReceiverSpan(t *testing.T, spans []sdktrace.ReadOnlySpan) {
	receiverSpan := false
	for _, span := range spans {
		if span.Name() == "receiver/receiver/LogsReceived" {
			receiverSpan = true
			break
		}
	}
	assert.True(t, receiverSpan)
}

func assertLogsReceiverViews(t *testing.T, tt componenttest.TestTelemetry, sink *consumertest.LogsSink) {
	logCount := 0
	for _, md := range sink.AllLogs() {
		logCount += md.LogRecordCount()
	}
	require.NoError(t, tt.CheckReceiverLogs("", int64(logCount), 0))
}

func assertLogsScraperSpan(t *testing.T, expectedErr error, spans []sdktrace.ReadOnlySpan) {
	expectedStatusCode := codes.Unset
	expectedStatusMessage := ""
	if expectedErr != nil {
		expectedStatusCode = codes.Error
		expectedStatusMessage = expectedErr.Error()
	}

	scraperSpan := false
	for _, span := range spans {
		if span.Name() == "scraper/scraper/ScrapeLogs" {
			scraperSpan = true
			assert.Equal(t, expectedStatusCode, span.Status().Code)
			assert.Equal(t, expectedStatusMessage, span.Status().Description)
			break
		}
	}
	assert.True(t, scraperSpan)
}

func assertLogsScraperViews(t *testing.T, tt componenttest.TestTelemetry, expectedErr error, sink *consumertest.LogsSink) {
	expectedScraped := int64(sink.LogRecordCount())
	expectedErrored := int64(0)
	if expectedErr != nil {
		var partialError scrapererror.PartialScrapeError
		if errors.As(expectedErr, &partialError) {
			expectedErrored = int64(partialError.Failed)
		} else {
			expectedScraped = int64(0)
			expectedErrored = int64(sink.LogRecordCount())
		}
	}

	require.NoError(t, tt.CheckScraperLogs(component.MustNewID("receiver"), component.MustNewID("scraper"), expectedScraped, expectedErrored))
}

func TestLogsSingleScrapePerInterval(t *testing.T) {
	scrapeLogsCh := make(chan int, 10)
	tsm := &testScrapeLogs{ch: scrapeLogsCh}

	cfg := newTestNoDelaySettings()

	tickerCh := make(chan time.Time)

	scp, err := scraper.NewLogs(tsm.scrape)
	require.NoError(t, err)

	recv, err := NewLogsScraperControllerReceiver(
		cfg,
		receivertest.NewNopSettings(),
		new(consumertest.LogsSink),
		AddLogsScraper(component.MustNewType("scaper"), scp),
		WithLogsTickerChannel(tickerCh),
	)
	require.NoError(t, err)

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, recv.Shutdown(context.Background())) }()

	tickerCh <- time.Now()

	assert.Eventually(
		t,
		func() bool {
			return <-scrapeLogsCh == 2
		},
		300*time.Millisecond,
		100*time.Millisecond,
		"Make sure the scraper channel is called twice",
	)

	select {
	case <-scrapeLogsCh:
		assert.Fail(t, "Scrape was called more than twice")
	case <-time.After(100 * time.Millisecond):
		return
	}
}

func TestLogsScrapeControllerStartsOnInit(t *testing.T) {
	t.Parallel()

	tsm := &testScrapeLogs{
		ch: make(chan int, 1),
	}

	scp, err := scraper.NewLogs(tsm.scrape)
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewLogsScraperControllerReceiver(
		&ControllerConfig{
			CollectionInterval: time.Hour,
			InitialDelay:       0,
		},
		receivertest.NewNopSettings(),
		new(consumertest.LogsSink),
		AddLogsScraper(component.MustNewType("scaper"), scp),
	)
	require.NoError(t, err, "Must not error when creating scrape controller")

	assert.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()), "Must not error on start")
	<-time.After(500 * time.Nanosecond)
	require.NoError(t, r.Shutdown(context.Background()), "Must not have errored on shutdown")
	assert.Equal(t, 1, tsm.timesScrapeCalled, "Must have been called as soon as the controller started")
}

func TestLogsScrapeControllerInitialDelay(t *testing.T) {
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

	scp, err := scraper.NewLogs(func(context.Context) (plog.Logs, error) {
		elapsed <- time.Now()
		return plog.NewLogs(), nil
	})
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewLogsScraperControllerReceiver(
		&cfg,
		receivertest.NewNopSettings(),
		new(consumertest.LogsSink),
		AddLogsScraper(component.MustNewType("scaper"), scp),
	)
	require.NoError(t, err, "Must not error when creating receiver")

	t0 := time.Now()
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()), "Must not error when starting")
	t1 := <-elapsed

	assert.GreaterOrEqual(t, t1.Sub(t0), 300*time.Millisecond, "Must have had 300ms pass as defined by initial delay")

	assert.NoError(t, r.Shutdown(context.Background()), "Must not error closing down")
}

func TestLogsShutdownBeforeScrapeCanStart(t *testing.T) {
	cfg := ControllerConfig{
		CollectionInterval: time.Second,
		InitialDelay:       5 * time.Second,
	}

	scp, err := scraper.NewLogs(func(context.Context) (plog.Logs, error) {
		// make the scraper wait for long enough it would disrupt a shutdown.
		time.Sleep(30 * time.Second)
		return plog.NewLogs(), nil
	})
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewLogsScraperControllerReceiver(
		&cfg,
		receivertest.NewNopSettings(),
		new(consumertest.LogsSink),
		AddLogsScraper(component.MustNewType("scaper"), scp),
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
