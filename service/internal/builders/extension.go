// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders // import "go.opentelemetry.io/collector/service/internal/builders"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

// Extension is an interface that allows using implementations of the builder
// from different packages.
type Extension interface {
	Create(context.Context, extension.Settings) (extension.Extension, error)
	Factory(component.Type) component.Factory
}

// ExtensionBuilder is a helper struct that given a set of Configs and Factories helps with creating extensions.
type ExtensionBuilder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]extension.Factory
}

// NewExtension creates a new ExtensionBuilder to help with creating
// components form a set of configs and factories.
func NewExtension(cfgs map[component.ID]component.Config, factories map[component.Type]extension.Factory) *ExtensionBuilder {
	return &ExtensionBuilder{cfgs: cfgs, factories: factories}
}

// Create creates an extension based on the settings and configs available.
func (b *ExtensionBuilder) Create(ctx context.Context, set extension.Settings) (extension.Extension, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("extension %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("extension factory not available for: %q", set.ID)
	}

	sl := f.Stability()
	if sl >= component.StabilityLevelAlpha {
		set.Logger.Debug(sl.LogMessage())
	} else {
		set.Logger.Info(sl.LogMessage())
	}
	return f.Create(ctx, set, cfg)
}

func (b *ExtensionBuilder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}

// NewNopExtensionConfigsAndFactories returns a configuration and factories that allows building a new nop processor.
func NewNopExtensionConfigsAndFactories() (map[component.ID]component.Config, map[component.Type]extension.Factory) {
	nopFactory := extensiontest.NewNopFactory()
	configs := map[component.ID]component.Config{
		component.NewID(NopType): nopFactory.CreateDefaultConfig(),
	}
	factories := map[component.Type]extension.Factory{
		NopType: nopFactory,
	}
	return configs, factories
}
