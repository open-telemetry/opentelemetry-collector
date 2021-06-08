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

package extensionhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// FactoryOption apply changes to ExporterOptions.
type FactoryOption func(o *factory)

// CreateDefaultConfig is the equivalent of component.ExtensionFactory.CreateDefaultConfig()
type CreateDefaultConfig func() config.Extension

// CreateServiceExtension is the equivalent of component.ExtensionFactory.CreateExtension()
type CreateServiceExtension func(context.Context, component.ExtensionCreateSettings, config.Extension) (component.Extension, error)

type factory struct {
	cfgType                config.Type
	createDefaultConfig    CreateDefaultConfig
	createServiceExtension CreateServiceExtension
}

// NewFactory returns a component.ExtensionFactory.
func NewFactory(
	cfgType config.Type,
	createDefaultConfig CreateDefaultConfig,
	createServiceExtension CreateServiceExtension,
	options ...FactoryOption) component.ExtensionFactory {
	f := &factory{
		cfgType:                cfgType,
		createDefaultConfig:    createDefaultConfig,
		createServiceExtension: createServiceExtension,
	}
	for _, opt := range options {
		opt(f)
	}
	return f
}

// Type gets the type of the Extension config created by this factory.
func (f *factory) Type() config.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *factory) CreateDefaultConfig() config.Extension {
	return f.createDefaultConfig()
}

// CreateExtension creates a component.TraceExtension based on this config.
func (f *factory) CreateExtension(
	ctx context.Context,
	set component.ExtensionCreateSettings,
	cfg config.Extension) (component.Extension, error) {
	return f.createServiceExtension(ctx, set, cfg)
}
