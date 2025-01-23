// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreate(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()

	// Create extension with a valid config.
	pCfg := cfg.(*Config)
	pCfg.MemoryLimitMiB = 5722
	pCfg.MemorySpikeLimitMiB = 1907
	pCfg.CheckInterval = 100 * time.Millisecond

	tp, err := factory.Create(context.Background(), extensiontest.NewNopSettings(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, tp)
	// test if we can shutdown a monitoring routine that has not started
	require.ErrorIs(t, tp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)
	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))

	assert.NoError(t, tp.Shutdown(context.Background()))
	// verify that no monitoring routine is running
	assert.ErrorIs(t, tp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)
}
