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
type Extension = component.Component

// ModuleInfo describes the go module for each component.
type ModuleInfo struct {
	Receiver  map[component.Type]string
	Processor map[component.Type]string
	Exporter  map[component.Type]string
	Extension map[component.Type]string
	Connector map[component.Type]string
}

// Settings is passed to Factory.Create(...) function.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo

	// ModuleInfo describes the go module for each component.
	ModuleInfo ModuleInfo
}

// CreateFunc is the equivalent of Factory.Create(...) function.
type CreateFunc func(context.Context, Settings, component.Config) (Extension, error)

// Create implements Factory.Create.
func (f CreateFunc) Create(ctx context.Context, set Settings, cfg component.Config) (Extension, error) {
	return f(ctx, set, cfg)
}

// Deprecated: [v0.112.0] use Create.
func (f CreateFunc) CreateExtension(ctx context.Context, set Settings, cfg component.Config) (Extension, error) {
	return f.Create(ctx, set, cfg)
}

type Factory interface {
	component.Factory

	// Create an extension based on the given config.
	Create(ctx context.Context, set Settings, cfg component.Config) (Extension, error)

	// Deprecated: [v0.112.0] use Create.
	CreateExtension(ctx context.Context, set Settings, cfg component.Config) (Extension, error)

	// Stability gets the stability level of the Extension.
	Stability() component.StabilityLevel

	// Deprecated: [v0.112.0] use Stability.
	ExtensionStability() component.StabilityLevel

	unexportedFactoryFunc()
}

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateFunc
	extensionStability component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) Stability() component.StabilityLevel {
	return f.extensionStability
}

func (f *factory) ExtensionStability() component.StabilityLevel {
	return f.extensionStability
}

// NewFactory returns a new Factory  based on this configuration.
func NewFactory(
	cfgType component.Type,
	createDefaultConfig component.CreateDefaultConfigFunc,
	createServiceExtension CreateFunc,
	sl component.StabilityLevel) Factory {
	return &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
		CreateFunc:              createServiceExtension,
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
