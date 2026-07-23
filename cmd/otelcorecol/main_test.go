// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/otelcol"
)

func TestNewCollectorSettingsIncludesEmbeddedSchema(t *testing.T) {
	set := newCollectorSettings()

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

func TestMain(t *testing.T) {
	originalRunCollector := runCollector
	originalExitProcess := exitProcess
	t.Cleanup(func() {
		runCollector = originalRunCollector
		exitProcess = originalExitProcess
	})

	runCalled := false
	exitCalled := false
	runCollector = func(set otelcol.CollectorSettings) error {
		runCalled = true
		require.Equal(t, embeddedConfigSchema, set.ConfigSchema)
		return nil
	}
	exitProcess = func(int) {
		exitCalled = true
	}

	main()

	require.True(t, runCalled)
	require.False(t, exitCalled)
}
