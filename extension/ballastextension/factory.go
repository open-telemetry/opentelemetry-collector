// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package ballastextension // import "go.opentelemetry.io/collector/extension/ballastextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/ballastextension/internal/metadata"
	"go.opentelemetry.io/collector/internal/iruntime"
)

// memHandler returns the total memory of the target host/vm
var memHandler = iruntime.TotalMemory

// NewFactory creates a factory for ballast extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(metadata.Type, createDefaultConfig, createExtension, metadata.ExtensionStability)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newMemoryBallast(cfg.(*Config), set.TelemetrySettings.Logger, memHandler), nil
}
