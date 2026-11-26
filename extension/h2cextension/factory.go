// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package h2cextension // import "go.opentelemetry.io/collector/extension/h2cextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/h2cextension/internal/metadata"
)

// NewFactory creates a factory for the h2c extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		create,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func create(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
	return &h2cExtension{}, nil
}
