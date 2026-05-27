// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCollectorSettingsIncludesEmbeddedSchema(t *testing.T) {
	set := newCollectorSettings()

	require.Equal(t, embeddedConfigSchema, set.ConfigSchema)
	require.Equal(t, "otelcorecol", set.BuildInfo.Command)
	require.Equal(t, "Local OpenTelemetry Collector binary, testing only.", set.BuildInfo.Description)
	require.NotNil(t, set.Factories)
}
