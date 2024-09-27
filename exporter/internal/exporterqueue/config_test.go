// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueConfig_Validate(t *testing.T) {
	qCfg := NewDefaultConfig()
	require.NoError(t, qCfg.Validate())

	qCfg.NumConsumers = 0
	require.EqualError(t, qCfg.Validate(), "number of consumers must be positive")

	qCfg = NewDefaultConfig()
	qCfg.QueueSize = 0
	require.EqualError(t, qCfg.Validate(), "queue size must be positive")

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	qCfg.Enabled = false
	assert.NoError(t, qCfg.Validate())
}
