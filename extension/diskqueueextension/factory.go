// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diskqueueextension // import "go.opentelemetry.io/collector/extension/diskqueueextension"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

const (
	TypeStr        = "disk_access"
	stabilityLevel = component.StabilityLevelDevelopment
)

func NewFactory() extension.Factory {
	return extension.NewFactory(component.MustNewType(TypeStr), createDefaultConfig, newExtension, stabilityLevel)
}

func newExtension(_ context.Context, settings extension.Settings, config component.Config) (extension.Extension, error) {
	return &diskAccessExtension{
		cfg:    config.(*Config),
		logger: settings.Logger,
	}, nil
}

func createDefaultConfig() component.Config {
	return &Config{
		MaxBytesPerFile:       10 * 1024 * 1024,
		SyncEvery:             1,
		SyncTimeout:           100 * time.Millisecond,
		MetadataTruncateEvery: 1000,
	}
}
