// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/memorylimiter"
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pipeline"
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

	core, observer := observer.New(zapcore.DebugLevel)
	attrs := attribute.NewSet(
		attribute.String(componentattribute.SignalKey, pipeline.SignalLogs.String()),
		attribute.String(componentattribute.ComponentIDKey, "memorylimiter"),
		attribute.String(componentattribute.PipelineIDKey, "logs/foo"),
	)
	set := processortest.NewNopSettingsWithType(factory.Type())
	set.Logger = componentattribute.NewLogger(zap.New(core), &attrs)

	tp, err := factory.CreateTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tp)
	// test if we can shutdown a monitoring routine that has not started
	require.ErrorIs(t, tp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)
	require.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))

	mp, err := factory.CreateMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, mp)
	require.NoError(t, mp.Start(context.Background(), componenttest.NewNopHost()))

	lp, err := factory.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, lp)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))

	assert.NoError(t, lp.Shutdown(context.Background()))
	assert.NoError(t, tp.Shutdown(context.Background()))
	assert.NoError(t, mp.Shutdown(context.Background()))
	// verify that no monitoring routine is running
	require.ErrorIs(t, tp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)

	// start and shutdown a new monitoring routine
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.Shutdown(context.Background()))
	// calling it again should throw an error
	require.ErrorIs(t, lp.Shutdown(context.Background()), memorylimiter.ErrShutdownNotStarted)

	var createLoggerCount int
	for _, log := range observer.All() {
		if log.Message == "created singleton logger" {
			createLoggerCount++
			assert.Empty(t, observer.All()[0].Context)
		}
	}
	assert.Equal(t, 1, createLoggerCount)
}
