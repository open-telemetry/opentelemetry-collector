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
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/internal/telemetry/telemetrytest"
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

	set := processortest.NewNopSettings(factory.Type())
	var droppedAttrs []string
	set.Logger = telemetrytest.MockInjectorLogger(set.Logger, &droppedAttrs)

	tp, err := factory.CreateTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tp)
	// test if we can shutdown a monitoring routine that has not started
	require.NoError(t, tp.Shutdown(context.Background()))
	require.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))

	mp, err := factory.CreateMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, mp)
	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	lp, err := factory.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, lp)
	require.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))

	pp, err := factory.CreateProfiles(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, pp)
	require.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))

	// Test that we've dropped the relevant injected attributes exactly once
	assert.ElementsMatch(t, droppedAttrs, []string{
		telemetry.SignalKey, telemetry.ComponentIDKey, telemetry.PipelineIDKey,
	})

	assert.NoError(t, lp.Shutdown(context.Background()))
	assert.NoError(t, tp.Shutdown(context.Background()))
	assert.NoError(t, mp.Shutdown(context.Background()))
	assert.NoError(t, pp.Shutdown(context.Background()))
	// verify that no monitoring routine is running
	require.NoError(t, tp.Shutdown(context.Background()))

	// start and shutdown a new monitoring routine
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.Shutdown(context.Background()))
	// calling it again should throw no error
	require.NoError(t, lp.Shutdown(context.Background()))
}
