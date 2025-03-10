// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package admissionlimiterextension // import "go.opentelemetry.io/collector/extension/admissionlimiterextension"

//go:generate mdatagen metadata.yaml

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/admissionlimiterextension/internal/metadata"
)

// NewFactory returns a new factory for the Memory Limiter extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		create,
		metadata.ExtensionStability)
}

// CreateDefaultConfig creates the default configuration for extension. Notice
// that the default configuration is expected to fail for this extension.
func createDefaultConfig() component.Config {
	return Config{
		RequestLimitMiB: 0,
		WaitingLimitMiB: 0,
	}
}

func create(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newAdmissionLimiter(cfg.(*Config), set.TelemetrySettings.Logger)
}
