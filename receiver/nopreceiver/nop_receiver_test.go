// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nopreceiver // import "go.opentelemetry.io/collector/receiver/nopreceiver"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewNopFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &struct{}{}, cfg)

	traces, err := factory.CreateTraces(context.Background(), receivertest.NewNopSettings(receivertest.NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, traces.Shutdown(context.Background()))

	metrics, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(receivertest.NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metrics.Shutdown(context.Background()))

	logs, err := factory.CreateLogs(context.Background(), receivertest.NewNopSettings(receivertest.NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logs.Shutdown(context.Background()))
}
