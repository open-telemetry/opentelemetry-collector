// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extension // import "go.opentelemetry.io/collector/extension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/componentalias"
)

// Extension is the interface for objects hosted by the OpenTelemetry Collector that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type Extension interface {
	component.Component
}

// Settings is passed to Factory.Create(...) function.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo

	// prevent unkeyed literal initialization
	_ struct{}
}

// CreateFunc is the equivalent of Factory.Create(...) function.
type CreateFunc func(context.Context, Settings, component.Config) (Extension, error)

type Factory interface {
	component.Factory

	// Create an extension based on the given config.
	Create(ctx context.Context, set Settings, cfg component.Config) (Extension, error)

	// Stability gets the stability level of the Extension.
	Stability() component.StabilityLevel

	unexportedFactoryFunc()
}

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	componentalias.TypeAliasHolder
	createFunc         CreateFunc
	extensionStability component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) Stability() component.StabilityLevel {
	return f.extensionStability
}

func (f *factory) Create(ctx context.Context, set Settings, cfg component.Config) (Extension, error) {
	if err := componentalias.ValidateComponentType(f, set.ID); err != nil {
		return nil, err
	}

	return f.createFunc(ctx, set, cfg)
}

// NewFactory returns a new Factory  based on this configuration.
func NewFactory(
	cfgType component.Type,
	createDefaultConfig component.CreateDefaultConfigFunc,
	createServiceExtension CreateFunc,
	sl component.StabilityLevel,
) Factory {
	return &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
		TypeAliasHolder:         componentalias.NewTypeAliasHolder(),
		createFunc:              createServiceExtension,
		extensionStability:      sl,
	}
}
