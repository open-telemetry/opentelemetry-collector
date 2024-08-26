// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package remotetapextension // import "go.opentelemetry.io/collector/extension/remotetapextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/remotetapextension/internal/metadata"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		component.StabilityLevelDevelopment,
	)
}

func createExtension(_ context.Context, settings extension.Settings, config component.Config) (extension.Extension, error) {
	cm := NewCallbackManager()
	return &remoteTapExtension{
		config:          config.(*Config),
		settings:        settings,
		callbackManager: cm,
		publisher:       NewPublisher(settings.Logger, cm),
	}, nil
}
