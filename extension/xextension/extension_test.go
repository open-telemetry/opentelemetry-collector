// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
)

type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
}

func TestWithDeprecatedTypeAlias(t *testing.T) {
	originalType := component.MustNewType("original")
	aliasType := component.MustNewType("alias")
	nopExtensionInstance := new(nopExtension)

	factory := NewFactory(
		originalType,
		func() component.Config { return &struct{}{} },
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return nopExtensionInstance, nil
		},
		component.StabilityLevelAlpha,
		WithDeprecatedTypeAlias(aliasType),
	)

	assert.Equal(t, originalType, factory.Type())

	ext, err := factory.Create(context.Background(), extension.Settings{
		ID:                component.NewID(originalType),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}, factory.CreateDefaultConfig())
	require.NoError(t, err)
	require.NotNil(t, ext)

	ext, err = factory.Create(context.Background(), extension.Settings{
		ID:                component.NewID(aliasType),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}, factory.CreateDefaultConfig())
	require.NoError(t, err)
	require.NotNil(t, ext)

	ext, err = factory.Create(context.Background(), extension.Settings{
		ID:                component.NewID(component.MustNewType("wrong")),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}, factory.CreateDefaultConfig())
	require.Error(t, err)
	require.Nil(t, ext)
}
