// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/service/internal/builders"
)

func TestExtensionBuilder(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	testID := component.NewID(testType)
	unknownID := component.MustNewID("unknown")

	factories, err := otelcol.MakeFactoryMap([]extension.Factory{
		extension.NewFactory(
			testType,
			func() component.Config { return &defaultCfg },
			func(_ context.Context, settings extension.Settings, _ component.Config) (extension.Extension, error) {
				return nopExtension{Settings: settings}, nil
			},
			component.StabilityLevelDevelopment),
	}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{testID: defaultCfg, unknownID: defaultCfg}
	b := builders.NewExtension(cfgs, factories)

	e, err := b.Create(context.Background(), createExtensionSettings(testID))
	require.NoError(t, err)
	assert.NotNil(t, e)

	// Check that the extension has access to the resource attributes.
	nop, ok := e.(nopExtension)
	assert.True(t, ok)
	assert.Equal(t, 0, nop.Settings.Resource.Attributes().Len())

	missingType, err := b.Create(context.Background(), createExtensionSettings(unknownID))
	require.EqualError(t, err, "extension factory not available for: \"unknown\"")
	assert.Nil(t, missingType)

	missingCfg, err := b.Create(context.Background(), createExtensionSettings(component.NewIDWithName(testType, "foo")))
	require.EqualError(t, err, "extension \"test/foo\" is not configured")
	assert.Nil(t, missingCfg)
}

func TestExtensionBuilderFactory(t *testing.T) {
	factories, err := otelcol.MakeFactoryMap([]extension.Factory{extension.NewFactory(component.MustNewType("foo"), nil, nil, component.StabilityLevelDevelopment)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := builders.NewExtension(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

func TestNewNopExtensionConfigsAndFactories(t *testing.T) {
	configs, factories := builders.NewNopExtensionConfigsAndFactories()
	builder := builders.NewExtension(configs, factories)
	require.NotNil(t, builder)

	factory := extensiontest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := extensiontest.NewNopSettings(factory.Type())
	set.ID = component.NewID(builders.NopType)

	ext, err := factory.Create(context.Background(), set, cfg)
	require.NoError(t, err)
	bExt, err := builder.Create(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, ext, bExt)
}

type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
	extension.Settings
}

func createExtensionSettings(id component.ID) extension.Settings {
	return extension.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
