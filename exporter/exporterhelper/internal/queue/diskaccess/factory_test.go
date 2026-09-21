// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diskaccess

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	require.NotNil(t, f)
	assert.Equal(t, component.MustNewType(TypeStr), f.Type())
	assert.Equal(t, component.StabilityLevelDevelopment, f.Stability())
}

func TestFactoryCreate(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig()
	settings := extensiontest.NewNopSettings(component.MustNewType(TypeStr))
	ext, err := f.Create(context.Background(), settings, cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
	dae, ok := ext.(*diskAccessExtension)
	require.True(t, ok)
	assert.Same(t, settings.Logger, dae.logger)
	assert.Same(t, cfg, dae.cfg)
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.Equal(t, &Config{
		MaxBytesPerFile:       10 * 1024 * 1024,
		SyncEvery:             1,
		SyncTimeout:           100 * time.Millisecond,
		MetadataTruncateEvery: 1000,
	}, cfg)
}
