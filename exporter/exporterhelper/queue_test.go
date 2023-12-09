// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueueConfig_Validate(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	assert.NoError(t, qCfg.Validate())

	qCfg.NumConsumers = 0
	assert.EqualError(t, qCfg.Validate(), "number of consumers must be positive")

	qCfg = NewDefaultQueueConfig()
	qCfg.QueueItemsSize = 0
	assert.EqualError(t, qCfg.Validate(), "queue size must be positive")

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	qCfg.Enabled = false
	assert.NoError(t, qCfg.Validate())
}
