// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configunmarshaler"

	"go.opentelemetry.io/collector/config"
)

type defaultConfigProvider struct {
	provider  configmapprovider.Provider
	merged    configmapprovider.Provider
	factories component.Factories
}

func NewDefaultConfigProvider(configFileName string, properties []string, factories component.Factories) configmapprovider.Provider {
	localProvider := configmapprovider.NewLocal(configFileName, properties)
	return &defaultConfigProvider{provider: localProvider, factories: factories}
}

func (mp *defaultConfigProvider) Retrieve(ctx context.Context, onChange func(event *configmapprovider.ChangeEvent)) (configmapprovider.Retrieved, error) {
	r, err := mp.provider.Retrieve(ctx, onChange)
	if err != nil {
		return nil, err
	}
	rootMap, err := r.Get(ctx)

	sources, err := unmarshalSources(ctx, rootMap, mp.factories)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	sources = append(sources, configmapprovider.NewSimple(rootMap))
	mp.merged = configmapprovider.NewMerge(sources...)

	return mp.merged.Retrieve(ctx, onChange)
}

func unmarshalSources(ctx context.Context, rootMap *config.Map, factories component.Factories) ([]configmapprovider.Provider, error) {
	// Unmarshal the "config_sources" section of rootMap and create a Provider for each
	// config source using the factories.ConfigSources.

	var providers []configmapprovider.Provider

	type RootConfig struct {
		ConfigSources []map[string]map[string]interface{} `mapstructure:"config_sources"`
	}
	var rootCfg RootConfig
	err := rootMap.Unmarshal(&rootCfg)
	if err != nil {
		return nil, err
	}

	for _, sourceCfg := range rootCfg.ConfigSources {
		for sourceType, settings := range sourceCfg {
			factory, ok := factories.ConfigSources[config.Type(sourceType)]
			if !ok {
				return nil, fmt.Errorf("unknown source type %q", sourceType)
			}

			cfg := factory.CreateDefaultConfig()
			cfg.SetIDName(sourceType)

			// Now that the default config struct is created we can Unmarshal into it,
			// and it will apply user-defined config on top of the default.
			if err := configunmarshaler.Unmarshal(config.NewMapFromStringMap(settings), cfg); err != nil {
				return nil, fmt.Errorf("error reading config of config source %q: %w", sourceType, err) // errorUnmarshalError(extensionsKeyName, id, err)
			}

			source, err := factory.CreateConfigSource(ctx, component.ConfigSourceCreateSettings{}, cfg)
			if err != nil {
				return nil, err
			}
			providers = append(providers, source)
		}
	}

	return providers, nil
}

func (mp *defaultConfigProvider) Shutdown(ctx context.Context) error {
	if mp.merged != nil {
		return mp.merged.Shutdown(ctx)
	}
	return nil
}
