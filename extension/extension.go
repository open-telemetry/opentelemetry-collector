// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extension // import "go.opentelemetry.io/collector/extension"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/internal"
)

// Extension is the interface for objects hosted by the OpenTelemetry Collector that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type Extension = internal.Extension

// Deprecated: [v0.109.0] Use [extensioncapabilities.Dependent] instead.
type Dependent = internal.Dependent

// Deprecated: [v0.109.0] Use [extensioncapabilities.PipelineWatcher] instead.
type PipelineWatcher = internal.PipelineWatcher

// Deprecated: [v0.109.0] Use [extensioncapabilities.ConfigWatcher] instead.
type ConfigWatcher = internal.ConfigWatcher

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

// CreateExtension implements Factory.Create.
func (f CreateFunc) CreateExtension(ctx context.Context, set Settings, cfg component.Config) (Extension, error) {
	return f(ctx, set, cfg)
}

type Factory interface {
	component.Factory

	// CreateExtension creates an extension based on the given config.
	CreateExtension(ctx context.Context, set Settings, cfg component.Config) (Extension, error)

	// ExtensionStability gets the stability level of the Extension.
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

// Builder extension is a helper struct that given a set of Configs and Factories helps with creating extensions.
//
// Deprecated: [v0.108.0] this builder is being internalized within the service module,
// and will be removed soon.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new extension.Builder to help with creating components form a set of configs and factories.
//
// Deprecated: [v0.108.0] this builder is being internalized within the service module,
// and will be removed soon.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// Create creates an extension based on the settings and configs available.
func (b *Builder) Create(ctx context.Context, set Settings) (Extension, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("extension %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("extension factory not available for: %q", set.ID)
	}

	sl := f.ExtensionStability()
	if sl >= component.StabilityLevelAlpha {
		set.Logger.Debug(sl.LogMessage())
	} else {
		set.Logger.Info(sl.LogMessage())
	}
	return f.CreateExtension(ctx, set, cfg)
}

func (b *Builder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}
