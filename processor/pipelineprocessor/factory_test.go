// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelineprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	config := cfg.(*Config)
	assert.Equal(t, defaultTimeout, config.TimeoutConfig.Timeout)
	assert.Equal(t, exporterhelper.NewDefaultQueueConfig(), config.QueueConfig)
	assert.Equal(t, configretry.NewDefaultBackOffConfig(), config.RetryConfig)
}

func TestCreateTracesProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	tp, err := factory.CreateTraces(context.Background(), processortest.NewNopSettings(factory.Type()), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, tp)
}

func TestCreateMetricsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	mp, err := factory.CreateMetrics(context.Background(), processortest.NewNopSettings(factory.Type()), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, mp)
}

func TestCreateLogsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	lp, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(factory.Type()), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, lp)
}
