// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsetOTLPEndpointEnvVars(t *testing.T) {
	// Save any existing env vars to restore after test
	originalEnvVars := map[string]string{}
	envVarNames := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	}
	for _, key := range envVarNames {
		if val, exists := os.LookupEnv(key); exists {
			originalEnvVars[key] = val
		}
	}
	// Restore original env vars after test
	defer func() {
		for _, key := range envVarNames {
			os.Unsetenv(key)
		}
		for key, val := range originalEnvVars {
			os.Setenv(key, val)
		}
	}()

	// Set up test env vars
	testValues := map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT":         "http://localhost:4317",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT":  "http://localhost:4318/v1/traces",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://localhost:4318/v1/metrics",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT":    "http://localhost:4318/v1/logs",
	}
	for key, val := range testValues {
		os.Setenv(key, val)
	}

	// Call the function
	saved := unsetOTLPEndpointEnvVars()

	// Verify env vars are unset
	for key := range testValues {
		_, exists := os.LookupEnv(key)
		assert.False(t, exists, "env var %s should be unset", key)
	}

	// Verify saved values match original values
	require.Len(t, saved, len(testValues))
	for key, expectedVal := range testValues {
		assert.Equal(t, expectedVal, saved[key], "saved value for %s should match", key)
	}
}

func TestUnsetOTLPEndpointEnvVars_PartiallySet(t *testing.T) {
	// Save any existing env vars to restore after test
	originalEnvVars := map[string]string{}
	envVarNames := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	}
	for _, key := range envVarNames {
		if val, exists := os.LookupEnv(key); exists {
			originalEnvVars[key] = val
		}
	}
	// Restore original env vars after test
	defer func() {
		for _, key := range envVarNames {
			os.Unsetenv(key)
		}
		for key, val := range originalEnvVars {
			os.Setenv(key, val)
		}
	}()

	// Clear all env vars first
	for _, key := range envVarNames {
		os.Unsetenv(key)
	}

	// Only set some env vars
	testValues := map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT":        "http://localhost:4317",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://localhost:4318/v1/traces",
	}
	for key, val := range testValues {
		os.Setenv(key, val)
	}

	// Call the function
	saved := unsetOTLPEndpointEnvVars()

	// Verify only the set env vars are saved
	require.Len(t, saved, len(testValues))
	for key, expectedVal := range testValues {
		assert.Equal(t, expectedVal, saved[key], "saved value for %s should match", key)
	}

	// Verify the env vars that weren't set are not in saved
	_, hasMetrics := saved["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"]
	_, hasLogs := saved["OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"]
	assert.False(t, hasMetrics, "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT should not be in saved")
	assert.False(t, hasLogs, "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT should not be in saved")
}

func TestRestoreEnvVars(t *testing.T) {
	// Save any existing env vars to restore after test
	originalEnvVars := map[string]string{}
	envVarNames := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
	}
	for _, key := range envVarNames {
		if val, exists := os.LookupEnv(key); exists {
			originalEnvVars[key] = val
		}
	}
	// Restore original env vars after test
	defer func() {
		for _, key := range envVarNames {
			os.Unsetenv(key)
		}
		for key, val := range originalEnvVars {
			os.Setenv(key, val)
		}
	}()

	// Clear env vars
	for _, key := range envVarNames {
		os.Unsetenv(key)
	}

	// Restore from a saved map
	savedValues := map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT":        "https://custom-endpoint:4317",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "https://traces-endpoint:4318",
	}
	restoreEnvVars(savedValues)

	// Verify env vars are restored
	for key, expectedVal := range savedValues {
		actualVal := os.Getenv(key)
		assert.Equal(t, expectedVal, actualVal, "env var %s should be restored", key)
	}
}

func TestRestoreEnvVars_EmptyMap(t *testing.T) {
	// Save any existing env var to restore after test
	origVal, origExists := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	defer func() {
		if origExists {
			os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", origVal)
		} else {
			os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		}
	}()

	// Set an env var
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://test:4317")

	// Restore with empty map should not change anything
	restoreEnvVars(map[string]string{})

	// Verify env var is unchanged
	assert.Equal(t, "http://test:4317", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
}

func TestUnsetAndRestoreEnvVars_RoundTrip(t *testing.T) {
	// Save any existing env vars to restore after test
	originalEnvVars := map[string]string{}
	envVarNames := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	}
	for _, key := range envVarNames {
		if val, exists := os.LookupEnv(key); exists {
			originalEnvVars[key] = val
		}
	}
	// Restore original env vars after test
	defer func() {
		for _, key := range envVarNames {
			os.Unsetenv(key)
		}
		for key, val := range originalEnvVars {
			os.Setenv(key, val)
		}
	}()

	// Set up test env vars
	testValues := map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT":         "https://production:4317",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT":  "https://traces.production:4318",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "https://metrics.production:4318",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT":    "https://logs.production:4318",
	}
	for key, val := range testValues {
		os.Setenv(key, val)
	}

	// Unset env vars
	saved := unsetOTLPEndpointEnvVars()

	// Verify they are unset
	for key := range testValues {
		_, exists := os.LookupEnv(key)
		require.False(t, exists, "env var %s should be unset after unsetOTLPEndpointEnvVars", key)
	}

	// Restore env vars
	restoreEnvVars(saved)

	// Verify they are restored to original values
	for key, expectedVal := range testValues {
		actualVal := os.Getenv(key)
		assert.Equal(t, expectedVal, actualVal, "env var %s should be restored to original value", key)
	}
}
