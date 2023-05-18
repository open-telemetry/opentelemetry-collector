// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ballastextension // import "go.opentelemetry.io/collector/extension/ballastextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/internal/iruntime"
)

const (
	// The value of extension "type" in configuration.
	typeStr = "memory_ballast"
)

// memHandler returns the total memory of the target host/vm
var memHandler = iruntime.TotalMemory

// NewFactory creates a factory for FluentBit extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(typeStr, createDefaultConfig, createExtension, component.StabilityLevelBeta)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createExtension(_ context.Context, set extension.CreateSettings, cfg component.Config) (extension.Extension, error) {
	return newMemoryBallast(cfg.(*Config), set.TelemetrySettings.Logger, memHandler), nil
}
