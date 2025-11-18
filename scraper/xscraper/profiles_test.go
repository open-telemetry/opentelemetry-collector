// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xscraper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestNewProfiles(t *testing.T) {
	mp, err := NewProfiles(newTestScrapeProfilesFunc(nil))
	require.NoError(t, err)

	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))
	md, err := mp.ScrapeProfiles(context.Background())
	require.NoError(t, err)
	assert.Equal(t, pprofile.NewProfiles(), md)
	require.NoError(t, mp.Shutdown(context.Background()))
}

func TestNewProfiles_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewProfiles(newTestScrapeProfilesFunc(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }))
	require.NoError(t, err)

	assert.Equal(t, want, mp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, mp.Shutdown(context.Background()))
}

func TestNewProfiles_NilRequiredFields(t *testing.T) {
	_, err := NewProfiles(nil)
	require.Error(t, err)
}

func TestNewProfiles_ProcessProfilesError(t *testing.T) {
	want := errors.New("my_error")
	mp, err := NewProfiles(newTestScrapeProfilesFunc(want))
	require.NoError(t, err)
	_, err = mp.ScrapeProfiles(context.Background())
	require.ErrorIs(t, err, want)
}

func TestProfilesConcurrency(t *testing.T) {
	mp, err := NewProfiles(newTestScrapeProfilesFunc(nil))
	require.NoError(t, err)
	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10000 {
				_, errScrape := mp.ScrapeProfiles(context.Background())
				assert.NoError(t, errScrape)
			}
		}()
	}
	wg.Wait()
	require.NoError(t, mp.Shutdown(context.Background()))
}

func newTestScrapeProfilesFunc(retError error) ScrapeProfilesFunc {
	return func(_ context.Context) (pprofile.Profiles, error) {
		return pprofile.NewProfiles(), retError
	}
}
