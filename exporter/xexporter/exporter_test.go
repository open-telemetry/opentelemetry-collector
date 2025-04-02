// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/internal/experr"
)

var testID = component.MustNewID("test")

func TestNewFactoryWithProfiles(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithProfiles(createProfiles, component.StabilityLevelDevelopment),
	)
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.ProfilesStability())
	_, err := factory.CreateProfiles(context.Background(), exporter.Settings{ID: testID}, &defaultCfg)
	require.NoError(t, err)

	wrongID := component.MustNewID("wrong")
	wrongIDErrStr := experr.ErrIDMismatch(wrongID, testType).Error()
	_, err = factory.CreateProfiles(context.Background(), exporter.Settings{ID: wrongID}, &defaultCfg)
	assert.EqualError(t, err, wrongIDErrStr)
}

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

// nop stores consumed profiles for testing purposes.
type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createProfiles(context.Context, exporter.Settings, component.Config) (Profiles, error) {
	return nopInstance, nil
}
