// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extension // import "go.opentelemetry.io/collector/component"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// Config is the configuration of a component.Extension. Specific Extension must implement
// this interface and must embed config.ExtensionSettings struct or a struct that extends it.
type Config interface {
	component.Config

	privateConfigExtension()
}

// Extension is the interface for objects hosted by the OpenTelemetry Collector that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type Extension interface {
	component.Component
}

// PipelineWatcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
	// Ready notifies the Extension that all pipelines were built and the
	// receivers were started, i.e.: the service is ready to receive data
	// (note that it may already have received data when this method is called).
	Ready() error

	// NotReady notifies the Extension that all receivers are about to be stopped,
	// i.e.: pipeline receivers will not accept new data.
	// This is sent before receivers are stopped, so the Extension can take any
	// appropriate actions before that happens.
	NotReady() error
}

// CreateSettings is passed to Factory.Create* functions.
type CreateSettings struct {
	Telemetry component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo
}

// CreateDefaultConfigFunc is the equivalent of Factory.CreateDefaultConfig()
type CreateDefaultConfigFunc func() Config

// CreateDefaultConfig implements Factory.CreateDefaultConfig()
func (f CreateDefaultConfigFunc) CreateDefaultConfig() Config {
	return f()
}

// CreateExtensionFunc is the equivalent of Factory.CreateExtension()
type CreateExtensionFunc func(context.Context, CreateSettings, Config) (Extension, error)

// CreateExtension implements Factory.CreateExtension.
func (f CreateExtensionFunc) CreateExtension(ctx context.Context, set CreateSettings, cfg Config) (Extension, error) {
	return f(ctx, set, cfg)
}

// Factory is a factory for extensions to the service.
type Factory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration for the Extension.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Extension.
	// The object returned by this method needs to pass the checks implemented by
	// 'componenttest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() Config

	// CreateExtension creates an extension based on the given config.
	CreateExtension(ctx context.Context, set CreateSettings, cfg Config) (Extension, error)

	// ExtensionStability gets the stability level of the Extension.
	ExtensionStability() component.StabilityLevel

	unexportedFunc()
}

type factory struct {
	cfgType component.Type
	CreateDefaultConfigFunc
	CreateExtensionFunc
	extensionStability component.StabilityLevel
}

func (f factory) Type() component.Type {
	return f.cfgType
}

func (f factory) unexportedFunc() {}

func (ef *factory) ExtensionStability() component.StabilityLevel {
	return ef.extensionStability
}

// NewExtensionFactory returns a new Factory  based on this configuration.
func NewExtensionFactory(
	cfgType component.Type,
	createDefaultConfig CreateDefaultConfigFunc,
	createServiceExtension CreateExtensionFunc,
	sl component.StabilityLevel) Factory {
	return &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
		CreateExtensionFunc:     createServiceExtension,
		extensionStability:      sl,
	}
}
