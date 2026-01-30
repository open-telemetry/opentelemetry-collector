// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"os"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func newSDK(ctx context.Context, res *sdkresource.Resource, conf config.OpenTelemetryConfiguration) (config.SDK, error) {
	resourceAttrs := make([]config.AttributeNameValue, 0, res.Len())
	for _, r := range res.Attributes() {
		key := string(r.Key)
		resourceAttrs = append(resourceAttrs, config.AttributeNameValue{
			Name:  key,
			Value: mustAttributeValueString(key, r.Value),
		})
	}
	conf.Resource = &config.Resource{
		SchemaUrl:  ptr(semconv.SchemaURL),
		Attributes: resourceAttrs,
	}

	// Temporarily unset OTEL_EXPORTER_OTLP_*_ENDPOINT environment variables to prevent
	// them from overriding the explicitly configured endpoints in the config file.
	// The otelconf package merges environment variables with the configuration, but
	// explicit configuration should take precedence.
	// See: https://github.com/open-telemetry/opentelemetry-collector/issues/14286
	savedEnvVars := unsetOTLPEndpointEnvVars()
	defer restoreEnvVars(savedEnvVars)

	return config.NewSDK(config.WithContext(ctx), config.WithOpenTelemetryConfiguration(conf))
}

// unsetOTLPEndpointEnvVars temporarily unsets OTLP endpoint environment variables
// and returns their original values for later restoration.
func unsetOTLPEndpointEnvVars() map[string]string {
	envVars := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	}

	saved := make(map[string]string)
	for _, key := range envVars {
		if value, exists := os.LookupEnv(key); exists {
			saved[key] = value
			os.Unsetenv(key)
		}
	}
	return saved
}

// restoreEnvVars restores environment variables to their original values.
func restoreEnvVars(saved map[string]string) {
	for key, value := range saved {
		os.Setenv(key, value)
	}
}
