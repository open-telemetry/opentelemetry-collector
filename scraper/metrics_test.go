// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestNewMetrics(t *testing.T) {
	mp, err := NewMetrics(newTestScrapeMetricsFunc(nil))
	require.NoError(t, err)

	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	md, err := mp.ScrapeMetrics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, pmetric.NewMetrics(), md)
	require.NoError(t, mp.Shutdown(context.Background()))
}

func TestNewMetrics_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetrics(newTestScrapeMetricsFunc(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }))
	require.NoError(t, err)

	assert.Equal(t, want, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, mp.Shutdown(context.Background()))
}

func TestNewMetrics_NilRequiredFields(t *testing.T) {
	_, err := NewMetrics(nil)
	require.Error(t, err)
}

func TestNewMetrics_ProcessMetricsError(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewMetrics(newTestScrapeMetricsFunc(want))
	require.NoError(t, err)
	_, err = mp.ScrapeMetrics(context.Background())
	require.ErrorIs(t, err, want)
}

func TestMetricsConcurrency(t *testing.T) {
	incomingMetrics := pmetric.NewMetrics()
	dps := incomingMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints()

	// Add 2 data points to the incoming
	dps.AppendEmpty()
	dps.AppendEmpty()

	mp, err := NewMetrics(newTestScrapeMetricsFunc(nil))
	require.NoError(t, err)
	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10000 {
				_, errScrape := mp.ScrapeMetrics(context.Background())
				assert.NoError(t, errScrape)
			}
		}()
	}
	wg.Wait()
	require.NoError(t, mp.Shutdown(context.Background()))
}

func newTestScrapeMetricsFunc(retError error) ScrapeMetricsFunc {
	return func(_ context.Context) (pmetric.Metrics, error) {
		return pmetric.NewMetrics(), retError
	}
}
