// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
	Settings
}

func TestNewFactory(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	nopExtensionInstance := new(nopExtension)

	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		func(context.Context, Settings, component.Config) (Extension, error) {
			return nopExtensionInstance, nil
		},
		component.StabilityLevelDevelopment)
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.Stability())
	ext, err := factory.Create(context.Background(), Settings{ID: component.NewID(testType)}, &defaultCfg)
	require.NoError(t, err)
	assert.Same(t, nopExtensionInstance, ext)

	_, err = factory.Create(context.Background(), Settings{ID: component.NewID(component.MustNewType("mismatch"))}, &defaultCfg)
	require.Error(t, err)
}
