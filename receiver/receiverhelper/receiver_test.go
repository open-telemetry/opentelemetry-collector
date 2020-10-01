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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

type testStart struct {
	ch  chan bool
	err error
}

func (ts *testStart) start(context.Context, component.Host) error {
	ts.ch <- true
	return ts.err
}

type testShutdown struct {
	ch  chan bool
	err error
}

func (ts *testShutdown) shutdown(context.Context) error {
	ts.ch <- true
	return ts.err
}

type baseTestCase struct {
	name        string
	start       bool
	shutdown    bool
	startErr    error
	shutdownErr error
}

func TestBaseReceiver(t *testing.T) {
	testCases := []baseTestCase{
		{
			name: "Standard",
		},
		{
			name:     "WithStartAndShutdown",
			start:    true,
			shutdown: true,
		},
		{
			name:        "WithStartAndShutdownErrors",
			start:       true,
			shutdown:    true,
			startErr:    errors.New("err1"),
			shutdownErr: errors.New("err2"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			startCh := make(chan bool, 1)
			shutdownCh := make(chan bool, 1)
			options := configureBaseOptions(test.start, test.shutdown, test.startErr, test.shutdownErr, startCh, shutdownCh)

			bp := newBaseReceiver("", options...)

			err := bp.Start(context.Background(), componenttest.NewNopHost())
			if test.startErr != nil {
				assert.Equal(t, test.startErr, err)
			} else {
				assert.NoError(t, err)
				if test.start {
					assertChannelCalled(t, startCh, "start was not called")
				}
			}

			err = bp.Shutdown(context.Background())
			if test.shutdownErr != nil {
				assert.Equal(t, test.shutdownErr, err)
			} else {
				assert.NoError(t, err)
				if test.shutdown {
					assertChannelCalled(t, shutdownCh, "shutdown was not called")
				}
			}
		})
	}
}

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

type testScrape struct {
	ch                chan int
	timesScrapeCalled int
	err               error
}

func (ts *testScrape) scrape(ctx context.Context) (pdata.Metrics, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled

	if ts.err != nil {
		return pdata.NewMetrics(), ts.err
	}

	return singleMetric(), nil
}

type metricsTestCase struct {
	name string

	start       bool
	shutdown    bool
	startErr    error
	shutdownErr error

	scrapers                  int
	defaultCollectionInterval time.Duration
	scraperSettings           ScraperSettings
	nilNextConsumer           bool
	scrapeErr                 error
	expectedNewErr            string
	expectScraped             bool

	initialize    bool
	close         bool
	initializeErr error
	closeErr      error
}

func TestMetricReceiver(t *testing.T) {
	testCases := []metricsTestCase{
		{
			name: "Standard",
		},
		{
			name:     "WithBaseOptions",
			start:    true,
			shutdown: true,
		},
		{
			name:        "WithBaseOptionsStartAndShutdownErrors",
			start:       true,
			shutdown:    true,
			startErr:    errors.New("err1"),
			shutdownErr: errors.New("err2"),
		},
		{
			name:                      "AddScrapersWithDefaultCollectionInterval",
			scrapers:                  2,
			defaultCollectionInterval: time.Millisecond,
			expectScraped:             true,
		},
		{
			name:            "AddScrapersWithCollectionInterval",
			scrapers:        2,
			scraperSettings: ScraperSettings{CollectionIntervalVal: time.Millisecond},
			expectScraped:   true,
		},
		{
			name:            "AddScrapers_NewError",
			scrapers:        2,
			nilNextConsumer: true,
			expectedNewErr:  "nil nextConsumer",
		},
		{
			name:                      "AddScrapers_ScrapeError",
			scrapers:                  2,
			defaultCollectionInterval: time.Millisecond,
			scrapeErr:                 errors.New("err1"),
		},
		{
			name:       "AddScrapersWithInitializeAndClose",
			scrapers:   2,
			initialize: true,
			close:      true,
		},
		{
			name:          "AddScrapersWithInitializeAndCloseErrors",
			scrapers:      2,
			initialize:    true,
			close:         true,
			initializeErr: errors.New("err1"),
			closeErr:      errors.New("err2"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			startCh := make(chan bool, 1)
			shutdownCh := make(chan bool, 1)
			baseOptions := configureBaseOptions(test.start, test.shutdown, test.startErr, test.shutdownErr, startCh, shutdownCh)

			initializeChs := make([]chan bool, test.scrapers)
			scrapeChs := make([]chan int, test.scrapers)
			closeChs := make([]chan bool, test.scrapers)
			options := configureMetricOptions(baseOptions, test, initializeChs, scrapeChs, closeChs)

			var nextConsumer consumer.MetricsConsumer
			sink := &exportertest.SinkMetricsExporter{}
			if !test.nilNextConsumer {
				nextConsumer = sink
			}
			mr, err := NewMetricReceiver(&configmodels.ReceiverSettings{}, nextConsumer, options...)
			if test.expectedNewErr != "" {
				assert.EqualError(t, err, test.expectedNewErr)
				return
			}
			require.NoError(t, err)

			err = mr.Start(context.Background(), componenttest.NewNopHost())
			expectedStartErr := getExpectedStartErr(test)
			if expectedStartErr != nil {
				assert.Equal(t, expectedStartErr, err)
			} else {
				if test.start {
					assertChannelCalled(t, startCh, "start was not called")
				}
				if test.initialize {
					assertChannelsCalled(t, initializeChs, "initialize was not called")
				}
			}

			// TODO: validate that observability information is reported correctly on error
			//       i.e. if test.scrapeErr != nil { ... }

			// validate that scrape is called at least 5 times for each configured scraper
			if test.expectScraped {
				for _, ch := range scrapeChs {
					require.Eventually(t, func() bool { return (<-ch) > 5 }, 500*time.Millisecond, time.Millisecond)
				}

				assert.Greater(t, sink.MetricsCount(), 5)
			}

			err = mr.Shutdown(context.Background())
			expectedShutdownErr := getExpectedShutdownErr(test)
			if expectedShutdownErr != nil {
				assert.EqualError(t, err, expectedShutdownErr.Error())
			} else {
				if test.shutdown {
					assertChannelCalled(t, shutdownCh, "shutdown was not called")
				}
				if test.close {
					assertChannelsCalled(t, closeChs, "clost was not called")
				}
			}
		})
	}
}

func configureBaseOptions(start, shutdown bool, startErr, shutdownErr error, startCh, shutdownCh chan bool) []Option {
	baseOptions := []Option{}
	if start {
		ts := &testStart{ch: startCh, err: startErr}
		baseOptions = append(baseOptions, WithStart(ts.start))
	}
	if shutdown {
		ts := &testShutdown{ch: shutdownCh, err: shutdownErr}
		baseOptions = append(baseOptions, WithShutdown(ts.shutdown))
	}
	return baseOptions
}

func configureMetricOptions(baseOptions []Option, test metricsTestCase, initializeChs []chan bool, scrapeChs []chan int, closeChs []chan bool) []MetricOption {
	metricOptions := []MetricOption{}
	metricOptions = append(metricOptions, WithBaseOptions(baseOptions...))

	if test.defaultCollectionInterval != 0 {
		metricOptions = append(metricOptions, WithDefaultCollectionInterval(test.defaultCollectionInterval))
	}

	for i := 0; i < test.scrapers; i++ {
		scraperOptions := []ScraperOption{}
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

		scrapeChs[i] = make(chan int, 10)
		ts := &testScrape{ch: scrapeChs[i], err: test.scrapeErr}
		metricOptions = append(metricOptions, AddScraper(&test.scraperSettings, ts.scrape, scraperOptions...))
	}

	return metricOptions
}

func getExpectedStartErr(test metricsTestCase) error {
	if test.startErr != nil {
		return test.startErr
	}

	return test.initializeErr
}

func getExpectedShutdownErr(test metricsTestCase) error {
	var errors []error

	if test.closeErr != nil {
		for i := 0; i < test.scrapers; i++ {
			errors = append(errors, test.closeErr)
		}
	}

	if test.shutdownErr != nil {
		errors = append(errors, test.shutdownErr)
	}

	return componenterror.CombineErrors(errors)
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
func singleMetric() pdata.Metrics {
	md := pdata.NewMetrics()
	rms := md.ResourceMetrics()
	rms.Resize(1)
	rm := rms.At(0)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ilm := ilms.At(0)
	metrics := ilm.Metrics()
	metrics.Resize(1)
	return md
}
