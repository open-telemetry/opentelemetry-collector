// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/memorylimiter"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()

	// Create processor with a valid config.
	pCfg := cfg.(*Config)
	pCfg.MemoryLimitMiB = 5722
	pCfg.MemorySpikeLimitMiB = 1907
	pCfg.CheckInterval = 100 * time.Millisecond

	tp, err := factory.CreateTracesProcessor(context.Background(), processortest.NewNopCreateSettings(), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, tp)
	// test if we can shutdown a monitoring routine that has not started
	assert.ErrorIs(t, tp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)
	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))

	mp, err := factory.CreateMetricsProcessor(context.Background(), processortest.NewNopCreateSettings(), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, mp)
	assert.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	lp, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, lp)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))

	assert.NoError(t, lp.Shutdown(context.Background()))
	assert.NoError(t, tp.Shutdown(context.Background()))
	assert.NoError(t, mp.Shutdown(context.Background()))
	// verify that no monitoring routine is running
	assert.ErrorIs(t, tp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)

	// start and shutdown a new monitoring routine
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.Shutdown(context.Background()))
	// calling it again should throw an error
	assert.ErrorIs(t, lp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)
}
