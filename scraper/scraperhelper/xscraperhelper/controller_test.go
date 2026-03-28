// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xscraperhelper provides utilities for scrapers.
package xscraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper"

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/testhelper"
	"go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/scraper/xscraper"
)

type testScrape struct {
	ch                chan int
	timesScrapeCalled int
	err               error
}

func (ts *testScrape) scrapeProfiles(context.Context) (pprofile.Profiles, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled

	if ts.err != nil {
		return pprofile.Profiles{}, ts.err
	}

	md := pprofile.NewProfiles()
	profile := md.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	profile.Samples().AppendEmpty()
	return md, nil
}

func newTestNoDelaySettings() *scraperhelper.ControllerConfig {
	return &scraperhelper.ControllerConfig{
		CollectionInterval: time.Second,
		InitialDelay:       0,
	}
}

type scraperTestCase struct {
	name string

	scrapers                  int
	scraperControllerSettings *scraperhelper.ControllerConfig
	scrapeErr                 error
	expectScraped             bool

	initialize    bool
	close         bool
	initializeErr error
	closeErr      error
}

func TestProfilesScrapeController(t *testing.T) {
	testCases := []scraperTestCase{
		{
			name: "NoScrapers",
		},
		{
			name:          "AddProfilesScrapersWithCollectionInterval",
			scrapers:      2,
			expectScraped: true,
		},
		{
			name:      "AddProfilesScrapers_ScrapeError",
			scrapers:  2,
			scrapeErr: errors.New("err1"),
		},
		{
			name:          "AddProfilesScrapersWithInitializeAndClose",
			scrapers:      2,
			initialize:    true,
			expectScraped: true,
			close:         true,
		},
		{
			name:          "AddProfilesScrapersWithInitializeAndCloseErrors",
			scrapers:      2,
			initialize:    true,
			close:         true,
			initializeErr: errors.New("err1"),
			closeErr:      errors.New("err2"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			receiverID := component.MustNewID("receiver")
			tel := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

			set := tel.NewTelemetrySettings()
			_, parentSpan := set.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
			defer parentSpan.End()

			initializeChs := make([]chan bool, test.scrapers)
			scrapeProfilesChs := make([]chan int, test.scrapers)
			closeChs := make([]chan bool, test.scrapers)
			options := configureProfilesOptions(t, test, initializeChs, scrapeProfilesChs, closeChs)

			tickerCh := make(chan time.Time)
			options = append(options, WithTickerChannel(tickerCh))

			sink := new(consumertest.ProfilesSink)
			cfg := newTestNoDelaySettings()
			if test.scraperControllerSettings != nil {
				cfg = test.scraperControllerSettings
			}

			mr, err := NewProfilesController(cfg, receiver.Settings{ID: receiverID, TelemetrySettings: set, BuildInfo: component.NewDefaultBuildInfo()}, sink, options...)
			require.NoError(t, err)

			err = mr.Start(context.Background(), componenttest.NewNopHost())
			expectedStartErr := getExpectedStartErr(test)
			if expectedStartErr != nil {
				assert.Equal(t, expectedStartErr, err)
			} else if test.initialize {
				testhelper.AssertChannelsCalled(t, initializeChs, "start was not called")
			}

			const iterations = 5

			if test.expectScraped || test.scrapeErr != nil {
				// validate that scrape is called at least N times for each configured scraper
				for _, ch := range scrapeProfilesChs {
					<-ch
				}
				// Consume the initial scrapes on start
				for range iterations {
					tickerCh <- time.Now()

					for _, ch := range scrapeProfilesChs {
						<-ch
					}
				}

				// wait until all calls to scrape have completed
				if test.scrapeErr == nil {
					require.Eventually(t, func() bool {
						return sink.SampleCount() == (1+iterations)*(test.scrapers)
					}, time.Second, time.Millisecond)
				}

				if test.expectScraped {
					assert.GreaterOrEqual(t, sink.SampleCount(), iterations)
				}

				spans := tel.SpanRecorder.Ended()
				testhelper.AssertScraperSpan(t, test.scrapeErr, spans, "scraper/scraper/ScrapeProfiles")
				assertProfilesScraperObsMetrics(t, tel, receiverID, component.MustNewID("scraper"), test.scrapeErr, sink)
			}

			err = mr.Shutdown(context.Background())
			expectedShutdownErr := getExpectedShutdownErr(test)
			if expectedShutdownErr != nil {
				assert.EqualError(t, err, expectedShutdownErr.Error())
			} else if test.close {
				testhelper.AssertChannelsCalled(t, closeChs, "shutdown was not called")
			}
		})
	}
}

func getExpectedStartErr(test scraperTestCase) error {
	return test.initializeErr
}

func getExpectedShutdownErr(test scraperTestCase) error {
	var errs []error

	if test.closeErr != nil {
		for i := 0; i < test.scrapers; i++ {
			errs = append(errs, test.closeErr)
		}
	}

	return multierr.Combine(errs...)
}

func configureProfilesOptions(t *testing.T, test scraperTestCase, initializeChs []chan bool, scrapeProfilesChs []chan int, closeChs []chan bool) []ControllerOption {
	var profilesOptions []ControllerOption

	for i := 0; i < test.scrapers; i++ {
		scrapeProfilesChs[i] = make(chan int)
		ts := &testScrape{ch: scrapeProfilesChs[i], err: test.scrapeErr}

		var xscraperOptions []xscraper.Option
		if test.initialize {
			initializeChs[i] = make(chan bool, 1)
			ti := testhelper.NewTestInitialize(initializeChs[i], test.initializeErr)
			xscraperOptions = append(xscraperOptions, xscraper.WithStart(ti.Start))
		}
		if test.close {
			closeChs[i] = make(chan bool, 1)
			tc := testhelper.NewTestClose(closeChs[i], test.closeErr)
			xscraperOptions = append(xscraperOptions, xscraper.WithShutdown(tc.Shutdown))
		}

		scp, err := xscraper.NewProfiles(ts.scrapeProfiles, xscraperOptions...)
		require.NoError(t, err)

		profilesOptions = append(profilesOptions, AddProfilesScraper(component.MustNewType("scraper"), scp))
	}

	return profilesOptions
}

func TestSingleProfilesScraperPerInterval(t *testing.T) {
	scrapeCh := make(chan int, 10)
	ts := &testScrape{ch: scrapeCh}

	cfg := newTestNoDelaySettings()

	tickerCh := make(chan time.Time)

	scp, err := xscraper.NewProfiles(ts.scrapeProfiles)
	require.NoError(t, err)

	recv, err := NewProfilesController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		new(consumertest.ProfilesSink),
		AddProfilesScraper(component.MustNewType("scraper"), scp),
		WithTickerChannel(tickerCh),
	)
	require.NoError(t, err)

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, recv.Shutdown(context.Background())) }()

	tickerCh <- time.Now()

	assert.Eventually(
		t,
		func() bool {
			return <-scrapeCh == 2
		},
		300*time.Millisecond,
		100*time.Millisecond,
		"Make sure the scraper channel is called twice",
	)

	select {
	case <-scrapeCh:
		assert.Fail(t, "Scrape was called more than twice")
	case <-time.After(100 * time.Millisecond):
		return
	}
}

func TestProfilesScraperControllerStartsOnInit(t *testing.T) {
	t.Parallel()

	ts := &testScrape{
		ch: make(chan int, 1),
	}

	scp, err := xscraper.NewProfiles(ts.scrapeProfiles)
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewProfilesController(
		&scraperhelper.ControllerConfig{
			CollectionInterval: time.Hour,
			InitialDelay:       0,
		},
		receivertest.NewNopSettings(receivertest.NopType),
		new(consumertest.ProfilesSink),
		AddProfilesScraper(component.MustNewType("scraper"), scp),
	)
	require.NoError(t, err, "Must not error when creating scrape controller")

	assert.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()), "Must not error on start")
	<-time.After(500 * time.Nanosecond)
	require.NoError(t, r.Shutdown(context.Background()), "Must not have errored on shutdown")
	assert.Equal(t, 1, ts.timesScrapeCalled, "Must have been called as soon as the controller started")
}

func TestProfilesScraperControllerInitialDelay(t *testing.T) {
	if testing.Short() {
		t.Skip("This requires real time to pass, skipping")
		return
	}

	t.Parallel()

	var (
		elapsed = make(chan time.Time, 1)
		cfg     = scraperhelper.ControllerConfig{
			CollectionInterval: time.Second,
			InitialDelay:       300 * time.Millisecond,
		}
	)

	scp, err := xscraper.NewProfiles(func(context.Context) (pprofile.Profiles, error) {
		elapsed <- time.Now()
		return pprofile.NewProfiles(), nil
	})
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewProfilesController(
		&cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		new(consumertest.ProfilesSink),
		AddProfilesScraper(component.MustNewType("scraper"), scp),
	)
	require.NoError(t, err, "Must not error when creating receiver")

	t0 := time.Now()
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()), "Must not error when starting")
	t1 := <-elapsed

	assert.GreaterOrEqual(t, t1.Sub(t0), 300*time.Millisecond, "Must have had 300ms pass as defined by initial delay")

	assert.NoError(t, r.Shutdown(context.Background()), "Must not error closing down")
}

func TestProfilesScraperShutdownBeforeScrapeCanStart(t *testing.T) {
	cfg := scraperhelper.ControllerConfig{
		CollectionInterval: time.Second,
		InitialDelay:       5 * time.Second,
	}

	scp, err := xscraper.NewProfiles(func(context.Context) (pprofile.Profiles, error) {
		// make the scraper wait for long enough it would disrupt a shutdown.
		time.Sleep(30 * time.Second)
		return pprofile.NewProfiles(), nil
	})
	require.NoError(t, err, "Must not error when creating scraper")

	r, err := NewProfilesController(
		&cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		new(consumertest.ProfilesSink),
		AddProfilesScraper(component.MustNewType("scraper"), scp),
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

func assertProfilesScraperObsMetrics(t *testing.T, tel *componenttest.Telemetry, receiver, scraper component.ID, expectedErr error, sink *consumertest.ProfilesSink) {
	sampleCounts := 0
	for _, md := range sink.AllProfiles() {
		sampleCounts += md.SampleCount()
	}

	expectedScraped := int64(sink.SampleCount())
	expectedErrored := int64(0)
	if expectedErr != nil {
		var partialError scrapererror.PartialScrapeError
		if errors.As(expectedErr, &partialError) {
			expectedErrored = int64(partialError.Failed)
		} else {
			expectedScraped = int64(0)
			expectedErrored = int64(sink.SampleCount())
		}
	}

	metadatatest.AssertEqualScraperScrapedProfileRecords(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(receiverKey, receiver.String()),
					attribute.String(scraperKey, scraper.String())),
				Value: expectedScraped,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	metadatatest.AssertEqualScraperErroredProfileRecords(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(receiverKey, receiver.String()),
					attribute.String(scraperKey, scraper.String())),
				Value: expectedErrored,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

// TestScrapeProfilesPreservesDictionaryMultipleScrapers verifies that the
// dictionary is correctly merged when multiple scrapers produce profiles
// with different dictionary entries.
func TestScrapeProfilesPreservesDictionaryMultipleScrapers(t *testing.T) {
	makeScraper := func(typeName, unitName string) func(context.Context) (pprofile.Profiles, error) {
		return func(context.Context) (pprofile.Profiles, error) {
			pd := pprofile.NewProfiles()
			dic := pd.Dictionary()
			dic.StringTable().Append("")       // index 0
			dic.StringTable().Append(typeName) // index 1
			dic.StringTable().Append(unitName) // index 2

			profile := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
			profile.SampleType().SetTypeStrindex(1)
			profile.SampleType().SetUnitStrindex(2)
			profile.Samples().AppendEmpty().Values().Append(1)

			return pd, nil
		}
	}

	tickerCh := make(chan time.Time)
	sink := new(consumertest.ProfilesSink)

	scpCPU, err := xscraper.NewProfiles(makeScraper("cpu", "nanoseconds"))
	require.NoError(t, err)
	scpHeap, err := xscraper.NewProfiles(makeScraper("alloc_space", "bytes"))
	require.NoError(t, err)

	recv, err := NewProfilesController(
		newTestNoDelaySettings(),
		receivertest.NewNopSettings(receivertest.NopType),
		sink,
		AddProfilesScraper(component.MustNewType("cpu_scraper"), scpCPU),
		AddProfilesScraper(component.MustNewType("heap_scraper"), scpHeap),
		WithTickerChannel(tickerCh),
	)
	require.NoError(t, err)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, recv.Shutdown(context.Background())) }()

	// Wait for the initial scrape
	require.Eventually(t, func() bool {
		return sink.SampleCount() >= 2
	}, time.Second, 10*time.Millisecond)

	profiles := sink.AllProfiles()
	require.NotEmpty(t, profiles)

	// The merged profiles should have both resource profiles and a unified dictionary
	pd := profiles[0]
	dic := pd.Dictionary()
	require.Equal(t, 2, pd.ResourceProfiles().Len(),
		"should have resource profiles from both scrapers")

	// Collect all strings in the merged dictionary
	allStrings := make(map[string]bool)
	for i := range dic.StringTable().Len() {
		allStrings[dic.StringTable().At(i)] = true
	}

	assert.True(t, allStrings["cpu"], "merged dictionary must contain 'cpu'")
	assert.True(t, allStrings["nanoseconds"], "merged dictionary must contain 'nanoseconds'")
	assert.True(t, allStrings["alloc_space"], "merged dictionary must contain 'alloc_space'")
	assert.True(t, allStrings["bytes"], "merged dictionary must contain 'bytes'")

	// Verify that SampleType indices resolve correctly for both profiles
	for i := range pd.ResourceProfiles().Len() {
		profile := pd.ResourceProfiles().At(i).ScopeProfiles().At(0).Profiles().At(0)
		st := profile.SampleType()
		typeName := dic.StringTable().At(int(st.TypeStrindex()))
		unitName := dic.StringTable().At(int(st.UnitStrindex()))
		assert.True(t, (typeName == "cpu" && unitName == "nanoseconds") ||
			(typeName == "alloc_space" && unitName == "bytes"),
			"SampleType must resolve to valid strings, got %s/%s", typeName, unitName)
	}
}

// TestNewProfilesControllerCreateError tests that NewProfilesController returns an error
// when the scraper factory's CreateProfiles method fails.
func TestNewProfilesControllerCreateError(t *testing.T) {
	expectedErr := errors.New("create profiles error")
	f := xscraper.NewFactory(component.MustNewType("scraper"), nil,
		xscraper.WithProfiles(func(context.Context, scraper.Settings, component.Config) (xscraper.Profiles, error) {
			return nil, expectedErr
		}, component.StabilityLevelDevelopment))

	cfg := newTestNoDelaySettings()
	_, err := NewProfilesController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		new(consumertest.ProfilesSink),
		AddFactoryWithConfig(f, nil),
	)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// errorMeter is a meter that returns errors when creating instruments.
type errorMeter struct {
	metric.Meter
}

func (errorMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errors.New("counter creation error")
}

// errorMeterProvider provides errorMeter instances.
type errorMeterProvider struct {
	metric.MeterProvider
}

func (errorMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return errorMeter{}
}

// TestNewProfilesControllerTelemetryError tests that NewProfilesController returns an error
// when telemetry builder creation fails.
func TestNewProfilesControllerTelemetryError(t *testing.T) {
	// Create a scraper that works
	scp, err := xscraper.NewProfiles(func(context.Context) (pprofile.Profiles, error) {
		return pprofile.NewProfiles(), nil
	})
	require.NoError(t, err)

	f := xscraper.NewFactory(component.MustNewType("scraper"), nil,
		xscraper.WithProfiles(func(context.Context, scraper.Settings, component.Config) (xscraper.Profiles, error) {
			return scp, nil
		}, component.StabilityLevelDevelopment))

	// Create telemetry settings with a meter provider that fails
	set := componenttest.NewNopTelemetrySettings()
	set.MeterProvider = errorMeterProvider{}

	cfg := newTestNoDelaySettings()
	_, err = NewProfilesController(
		cfg,
		receiver.Settings{
			ID:                component.MustNewID("receiver"),
			TelemetrySettings: set,
			BuildInfo:         component.NewDefaultBuildInfo(),
		},
		new(consumertest.ProfilesSink),
		AddFactoryWithConfig(f, nil),
	)

	// The error should be from wrapObsProfiles failing due to telemetry builder creation
	require.Error(t, err)
	assert.Contains(t, err.Error(), "counter creation error")
}
