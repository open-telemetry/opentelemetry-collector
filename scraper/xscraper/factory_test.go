// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/scraper"
)

var testType = component.MustNewType("test")

func nopSettings() scraper.Settings {
	return scraper.Settings{
		ID:                component.NewID(testType),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}
}

func TestNewFactory(t *testing.T) {
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg })
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())
	_, err := f.CreateProfiles(context.Background(), nopSettings(), &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
}

func TestNewFactoryWithOptions(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithLogs(createLogs, component.StabilityLevelAlpha),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithProfiles(createProfiles, component.StabilityLevelDevelopment))
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, f.ProfilesStability())
	_, err := f.CreateProfiles(context.Background(), scraper.Settings{}, &defaultCfg)
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, f.LogsStability())
	_, err = f.CreateLogs(context.Background(), scraper.Settings{}, &defaultCfg)
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, f.MetricsStability())
	_, err = f.CreateMetrics(context.Background(), scraper.Settings{}, &defaultCfg)
	require.NoError(t, err)
}

func createProfiles(context.Context, scraper.Settings, component.Config) (Profiles, error) {
	return NewProfiles(newTestScrapeProfilesFunc(nil))
}

func createLogs(context.Context, scraper.Settings, component.Config) (scraper.Logs, error) {
	return scraper.NewLogs(newTestScrapeLogsFunc(nil))
}

func createMetrics(context.Context, scraper.Settings, component.Config) (scraper.Metrics, error) {
	return scraper.NewMetrics(newTestScrapeMetricsFunc(nil))
}

func newTestScrapeLogsFunc(retError error) scraper.ScrapeLogsFunc {
	return func(_ context.Context) (plog.Logs, error) {
		return plog.NewLogs(), retError
	}
}

func newTestScrapeMetricsFunc(retError error) scraper.ScrapeMetricsFunc {
	return func(_ context.Context) (pmetric.Metrics, error) {
		return pmetric.NewMetrics(), retError
	}
}
