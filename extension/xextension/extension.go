// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xextension // import "go.opentelemetry.io/collector/extension/xextension"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/internal/componentalias"
)

type Factory interface {
	extension.Factory
}

type FactoryOption interface {
	applyOption(o *factory)
}

type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
}

type factory struct {
	extension.Factory
	componentalias.TypeAliasHolder
}

func WithDeprecatedTypeAlias(alias component.Type) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.SetDeprecatedAlias(alias)
	})
}

func NewFactory(
	cfgType component.Type,
	createDefaultConfig component.CreateDefaultConfigFunc,
	createServiceExtension extension.CreateFunc,
	sl component.StabilityLevel,
	options ...FactoryOption,
) Factory {
	f := &factory{TypeAliasHolder: componentalias.NewTypeAliasHolder()}
	for _, opt := range options {
		opt.applyOption(f)
	}
	f.Factory = extension.NewFactory(cfgType, createDefaultConfig, createServiceExtension, sl)
	f.Factory.(componentalias.TypeAliasHolder).SetDeprecatedAlias(f.DeprecatedAlias())
	return f
}
