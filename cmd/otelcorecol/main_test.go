// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/otelcol"
)

func TestNewDefaultCollectorSettingsIncludesEmbeddedSchema(t *testing.T) {
	set := newDefaultCollectorSettings()

	// The builder embeds the combined schema of the components that ship a
	// resolved config.schema.json (for example the OTLP receiver).
	require.NotEmpty(t, embeddedConfigSchema)
	require.True(t, json.Valid(embeddedConfigSchema))
	require.Equal(t, embeddedConfigSchema, set.ConfigSchema)
	require.Equal(t, "otelcorecol", set.BuildInfo.Command)
	require.Equal(t, "Local OpenTelemetry Collector binary, testing only.", set.BuildInfo.Description)
	require.NotNil(t, set.Factories)
}

func TestRunMainSuccess(t *testing.T) {
	var got otelcol.CollectorSettings
	exitCalled := false

	runMain(func(set otelcol.CollectorSettings) error {
		got = set
		return nil
	}, func(int) {
		exitCalled = true
	})

	require.False(t, exitCalled)
	require.Equal(t, embeddedConfigSchema, got.ConfigSchema)
}

func TestRunMainError(t *testing.T) {
	exitCode := 0

	runMain(func(otelcol.CollectorSettings) error {
		return errors.New("boom")
	}, func(code int) {
		exitCode = code
	})

	require.Equal(t, 1, exitCode)
}
