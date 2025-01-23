// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestFactory_CreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.Equal(t, &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:55679",
		},
	},
		cfg)

	require.NoError(t, componenttest.CheckConfigStruct(cfg))
	ext, err := create(context.Background(), extensiontest.NewNopSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestFactoryCreate(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.ServerConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	ext, err := create(context.Background(), extensiontest.NewNopSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
}
