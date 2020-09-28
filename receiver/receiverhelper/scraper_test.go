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
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

var testCfg = &ScraperSettings{
	ReceiverSettings: configmodels.ReceiverSettings{
		TypeVal: testFullName,
		NameVal: testFullName,
	},
	CollectionIntervalVal: time.Microsecond,
}

func TestNewScraper(t *testing.T) {
	var startCalled, shutdownCalled bool
	start := func(context.Context, component.Host) error { startCalled = true; return nil }
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	ch := make(chan int, 10)

	ts, err := NewScraper(testCfg, &testScraper{ch: ch}, exportertest.NewNopMetricsExporter(), WithStart(start), WithShutdown(shutdown))
	require.NoError(t, err)

	assert.NoError(t, ts.Start(context.Background(), componenttest.NewNopHost()))
	assert.True(t, startCalled)

	require.Eventually(t, func() bool { return (<-ch) > 5 }, 100*time.Millisecond, time.Millisecond)

	assert.NoError(t, ts.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

type testScraper struct {
	ch                chan int
	timesScrapeCalled int
}

func (ts *testScraper) Scrape(ctx context.Context) (pdata.Metrics, error) {
	ts.timesScrapeCalled++
	ts.ch <- ts.timesScrapeCalled
	return pdata.NewMetrics(), nil
}

func TestNewScraper_ScrapeError(t *testing.T) {
	want := errors.New("my_error")
	ts, err := NewScraper(testCfg, &testScraperError{err: want}, exportertest.NewNopMetricsExporter())
	require.NoError(t, err)

	assert.NoError(t, ts.Start(context.Background(), componenttest.NewNopHost()))
	// TODO validate observability data populated correctly
	assert.NoError(t, ts.Shutdown(context.Background()))
}

type testScraperError struct {
	err error
}

func (ts *testScraperError) Scrape(ctx context.Context) (pdata.Metrics, error) {
	return pdata.NewMetrics(), ts.err
}

func TestNewScraper_Error(t *testing.T) {
	_, err := NewScraper(testCfg, nil, nil)
	assert.Equal(t, componenterror.ErrNilNextConsumer, err)
}
