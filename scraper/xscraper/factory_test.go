// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/scraper"
)

func TestNewFactoryWithOptions(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithProfiles(createProfiles, component.StabilityLevelDevelopment))
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, f.ProfilesStability())
	_, err := f.CreateProfiles(context.Background(), scraper.Settings{}, &defaultCfg)
	require.NoError(t, err)
}

func createProfiles(context.Context, scraper.Settings, component.Config) (Profiles, error) {
	return NewProfiles(newTestScrapeProfilesFunc(nil))
}
