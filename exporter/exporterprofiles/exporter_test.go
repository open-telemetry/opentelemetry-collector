// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterprofiles

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
)

func TestNewFactoryWithProfiles(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithProfiles(createProfiles, component.StabilityLevelDevelopment),
	)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.ProfilesExporterStability())
	_, err := factory.CreateProfilesExporter(context.Background(), exporter.Settings{}, &defaultCfg)
	assert.NoError(t, err)
}

var nopInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

// nopExporter stores consumed profiles for testing purposes.
type nopExporter struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createProfiles(context.Context, exporter.Settings, component.Config) (Profiles, error) {
	return nopInstance, nil
}
