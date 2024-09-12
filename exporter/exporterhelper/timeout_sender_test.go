// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultTimeoutConfig(t *testing.T) {
	cfg := NewDefaultTimeoutConfig()
	assert.NoError(t, cfg.Validate())
	assert.Equal(t, TimeoutConfig{Timeout: 5 * time.Second}, cfg)
}

func TestInvalidTimeout(t *testing.T) {
	cfg := NewDefaultTimeoutConfig()
	assert.NoError(t, cfg.Validate())
	cfg.Timeout = -1
	assert.Error(t, cfg.Validate())
}
