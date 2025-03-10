// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

func TestNewNopFactory(t *testing.T) {
	factory := NewNopFactory()
	require.NotNil(t, factory)
	assert.Equal(t, "nop", factory.Type().String())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopConfig{}, cfg)

	traces, err := factory.CreateTraces(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, traces.Shutdown(context.Background()))

	metrics, err := factory.CreateMetrics(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metrics.Shutdown(context.Background()))

	logs, err := factory.CreateLogs(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logs.Shutdown(context.Background()))

	profiles, err := factory.(xreceiver.Factory).CreateProfiles(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, profiles.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, profiles.Shutdown(context.Background()))
}
