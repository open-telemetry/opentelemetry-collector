// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensiontest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
)

func TestStatusWatcherExtension(t *testing.T) {
	statusChanged := false
	factory := NewStatusWatcherExtensionFactory(
		func(*component.InstanceID, *component.StatusEvent) {
			statusChanged = true
		},
	)
	require.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("statuswatcher"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &struct{}{}, cfg)

	ext, err := factory.CreateExtension(context.Background(), NewStatusWatcherExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	assert.False(t, statusChanged)

	ext.(extension.StatusWatcher).ComponentStatusChanged(&component.InstanceID{}, &component.StatusEvent{})

	assert.True(t, statusChanged)
	assert.NoError(t, ext.Shutdown(context.Background()))
}
