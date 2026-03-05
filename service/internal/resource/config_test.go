// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

var buildInfo = component.BuildInfo{
	Command: "otelcol",
	Version: "1.0.0",
}

func TestDefaultAttributeValues(t *testing.T) {
	t.Run("defaults included", func(t *testing.T) {
		defaults, err := DefaultAttributeValues(buildInfo, nil)
		require.NoError(t, err)
		assert.Equal(t, buildInfo.Command, defaults["service.name"])
		assert.Equal(t, buildInfo.Version, defaults["service.version"])
		_, ok := defaults["service.instance.id"]
		assert.True(t, ok)
	})

	t.Run("defaults removed", func(t *testing.T) {
		removed := map[string]struct{}{
			"service.name":        {},
			"service.version":     {},
			"service.instance.id": {},
		}
		defaults, err := DefaultAttributeValues(buildInfo, removed)
		require.NoError(t, err)
		assert.NotContains(t, defaults, "service.name")
		assert.NotContains(t, defaults, "service.version")
		assert.NotContains(t, defaults, "service.instance.id")
	})

	t.Run("uuid failure", func(t *testing.T) {
		orig := newUUID
		t.Cleanup(func() { newUUID = orig })
		newUUID = func() (uuid.UUID, error) {
			return uuid.UUID{}, assert.AnError
		}

		_, err := DefaultAttributeValues(buildInfo, nil)
		require.ErrorContains(t, err, "failed to generate instance ID")
	})
}
