// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"sync/atomic"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGate(t *testing.T) {
	enabled := &atomic.Bool{}
	enabled.Store(true)
	from, err := version.NewVersion("v0.61.0")
	require.NoError(t, err)
	to, err := version.NewVersion("v0.64.0")
	require.NoError(t, err)

	g := &Gate{
		id:           "test",
		description:  "test gate",
		enabled:      enabled,
		stage:        StageAlpha,
		referenceURL: "http://example.com",
		fromVersion:  from,
		toVersion:    to,
	}

	assert.Equal(t, "test", g.ID())
	assert.Equal(t, "test gate", g.Description())
	assert.True(t, g.IsEnabled())
	assert.Equal(t, StageAlpha, g.Stage())
	assert.Equal(t, "http://example.com", g.ReferenceURL())
	assert.Equal(t, "v0.61.0", g.FromVersion())
	assert.Equal(t, "v0.64.0", g.ToVersion())
}
