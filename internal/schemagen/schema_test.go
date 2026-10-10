// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromMetadata_Deprecated(t *testing.T) {
	md := &ConfigsMetadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"endpoint": {Type: "string"},
				"interval": {
					Type:       "string",
					Deprecated: &DeprecatedConfig{Since: "v0.160.0", Note: "will be removed"},
				},
			},
		},
	}

	jsonSchema := FromMetadata("test", "test", md)

	require.NotNil(t, jsonSchema.Properties["endpoint"])
	assert.False(t, jsonSchema.Properties["endpoint"].Deprecated)
	require.NotNil(t, jsonSchema.Properties["interval"])
	assert.True(t, jsonSchema.Properties["interval"].Deprecated)
}
