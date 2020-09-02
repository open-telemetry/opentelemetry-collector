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

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// FactoryOption apply changes to ExporterOptions.
type FactoryOption func(o *factory)

// CreateDefaultConfig is the equivalent of component.ExtensionFactory.CreateDefaultConfig()
type CreateDefaultConfig func() configmodels.Extension

// CreateServiceExtension is the equivalent of component.ExtensionFactory.CreateExtension()
type CreateServiceExtension func(context.Context, component.ExtensionCreateParams, configmodels.Extension) (component.ServiceExtension, error)

type factory struct {
	cfgType                configmodels.Type
	customUnmarshaler      component.CustomUnmarshaler
	createDefaultConfig    CreateDefaultConfig
	createServiceExtension CreateServiceExtension
}

// WithCustomUnmarshaler implements component.ConfigUnmarshaler.
func WithCustomUnmarshaler(customUnmarshaler component.CustomUnmarshaler) FactoryOption {
	return func(o *factory) {
		o.customUnmarshaler = customUnmarshaler
	}
}

// NewFactory returns a component.ExtensionFactory.
func NewFactory(
	cfgType configmodels.Type,
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
	var ret component.ExtensionFactory
	if f.customUnmarshaler != nil {
		ret = &factoryWithUnmarshaler{f}
	} else {
		ret = f
	}
	return ret
}

// Type gets the type of the Extension config created by this factory.
func (f *factory) Type() configmodels.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *factory) CreateDefaultConfig() configmodels.Extension {
	return f.createDefaultConfig()
}

// CreateExtension creates a component.TraceExtension based on this config.
func (f *factory) CreateExtension(
	ctx context.Context,
	params component.ExtensionCreateParams,
	cfg configmodels.Extension) (component.ServiceExtension, error) {
	return f.createServiceExtension(ctx, params, cfg)
}

var _ component.ConfigUnmarshaler = (*factoryWithUnmarshaler)(nil)

type factoryWithUnmarshaler struct {
	*factory
}

// Unmarshal un-marshals the config using the provided custom unmarshaler.
func (f *factoryWithUnmarshaler) Unmarshal(componentViperSection *viper.Viper, intoCfg interface{}) error {
	return f.customUnmarshaler(componentViperSection, intoCfg)
}
