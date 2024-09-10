// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

var nopType = component.MustNewType("nop")

func TestNewNopFactory(t *testing.T) {
	factory := NewNopFactory()
	require.NotNil(t, factory)
	assert.Equal(t, nopType, factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopConfig{}, cfg)

	traces, err := factory.CreateTracesReceiver(context.Background(), NewNopSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, traces.Shutdown(context.Background()))

	metrics, err := factory.CreateMetricsReceiver(context.Background(), NewNopSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metrics.Shutdown(context.Background()))

	logs, err := factory.CreateLogsReceiver(context.Background(), NewNopSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logs.Shutdown(context.Background()))

	profiles, err := factory.CreateProfilesReceiver(context.Background(), NewNopSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, profiles.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, profiles.Shutdown(context.Background()))
}
