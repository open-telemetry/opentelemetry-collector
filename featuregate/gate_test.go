// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGate(t *testing.T) {
	enabled := &atomic.Bool{}
	enabled.Store(true)
	g := &Gate{
		id:           "test",
		description:  "test gate",
		enabled:      enabled,
		stage:        StageAlpha,
		referenceURL: "http://example.com",
		fromVersion:  "v0.61.0",
		toVersion:    "v0.64.0",
	}

	assert.Equal(t, "test", g.ID())
	assert.Equal(t, "test gate", g.Description())
	assert.True(t, g.IsEnabled())
	assert.Equal(t, StageAlpha, g.Stage())
	assert.Equal(t, "http://example.com", g.ReferenceURL())
	assert.Equal(t, "v0.61.0", g.FromVersion())
	assert.Equal(t, "v0.64.0", g.ToVersion())
}
