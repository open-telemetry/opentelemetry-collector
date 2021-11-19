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

package defaultconfigprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configunmarshaler"

	"go.opentelemetry.io/collector/config"
)

type defaultConfigProvider struct {
	initial   configmapprovider.ConfigSource
	merged    configmapprovider.ConfigSource
	factories component.Factories
}

func NewDefaultConfigProvider(configFileName string, properties []string, factories component.Factories) configmapprovider.ConfigSource {
	localProvider := configmapprovider.NewLocal(configFileName, properties)
	return &defaultConfigProvider{initial: localProvider, factories: factories}
}

func (mp *defaultConfigProvider) Retrieve(ctx context.Context, onChange func(event *configmapprovider.ChangeEvent)) (configmapprovider.RetrievedConfig, error) {
	r, err := mp.initial.Retrieve(ctx, onChange)
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

	retrieved, err := mp.merged.Retrieve(ctx, onChange)
	if err != nil {
		return nil, fmt.Errorf("cannot retrive the configuration: %w", err)
	}

	return &valueSourceSubstitutor{onChange: onChange, retrieved: retrieved}, nil
}

func unmarshalSources(ctx context.Context, rootMap *config.Map, factories component.Factories) ([]configmapprovider.ConfigSource, error) {
	// Unmarshal the "config_sources" section of rootMap and create a Shutdownable for each
	// config source using the factories.ConfigSources.

	var providers []configmapprovider.ConfigSource

	type RootConfig struct {
		ConfigSources []map[string]map[string]interface{} `mapstructure:"config_sources"`
		ValueSources  []map[string]map[string]interface{} `mapstructure:"value_sources"`
	}
	var rootCfg RootConfig
	err := rootMap.Unmarshal(&rootCfg)
	if err != nil {
		return nil, err
	}

	for _, sourceCfg := range rootCfg.ConfigSources {
		for sourceType, settings := range sourceCfg {
			factoryBase, ok := factories.ConfigSources[config.Type(sourceType)]
			if !ok {
				return nil, fmt.Errorf("unknown source type %q", sourceType)
			}

			cfg := factoryBase.CreateDefaultConfig()
			cfg.SetIDName(sourceType)

			// Now that the default config struct is created we can Unmarshal into it,
			// and it will apply user-defined config on top of the default.
			if err := configunmarshaler.Unmarshal(config.NewMapFromStringMap(settings), cfg); err != nil {
				return nil, fmt.Errorf("error reading config of config source %q: %w", sourceType, err)
			}

			factory, ok := factoryBase.(component.ConfigSourceFactory)
			if !ok {
				return nil, fmt.Errorf("config source %q does not implement ConfigSourceFactory", sourceType)
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
