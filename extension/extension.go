// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extension // import "go.opentelemetry.io/collector/extension"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
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
}

// CreateFunc is the equivalent of Factory.Create(...) function.
type CreateFunc func(context.Context, Settings, component.Config) (Extension, error)

// Create implements Factory.Create.
// Deprecated: [v0.120.0] No longer used, will be removed.
func (f CreateFunc) Create(ctx context.Context, set Settings, cfg component.Config) (Extension, error) {
	return f(ctx, set, cfg)
}

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
	if set.ID.Type() != f.cfgType {
		return nil, fmt.Errorf("component type mismatch: component ID %q does not have type %q", set.ID, f.cfgType)
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
		createFunc:              createServiceExtension,
		extensionStability:      sl,
	}
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate extension factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
