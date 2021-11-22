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
	initial   configmapprovider.Provider
	merged    configmapprovider.Provider
	factories component.Factories
}

func NewDefaultConfigProvider(configFileName string, properties []string, factories component.Factories) configmapprovider.Provider {
	localProvider := configmapprovider.NewLocal(configFileName, properties)
	return &defaultConfigProvider{initial: localProvider, factories: factories}
}

func (mp *defaultConfigProvider) Retrieve(ctx context.Context, onChange func(event *configmapprovider.ChangeEvent)) (configmapprovider.RetrievedConfig, error) {
	r, err := mp.initial.Retrieve(ctx, onChange)
	if err != nil {
		return nil, err
	}
	rootMap, err := r.Get(ctx)

	configSources, mergeConfigs, err := unmarshalSources(ctx, rootMap, mp.factories)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	var rootProviders []configmapprovider.Provider
	for _, configSource := range mergeConfigs {
		configSourceID, err := config.NewComponentIDFromString(configSource)
		if err != nil {
			return nil, err
		}

		configSource, ok := configSources[configSourceID]
		if !ok {
			return nil, fmt.Errorf("config source %q must be defined in config_sources section", configSourceID)
		}

		mapProvider, ok := configSource.(configmapprovider.Provider)
		if !ok {
			return nil, fmt.Errorf("config source %q cannot be used in merge_configs section since it does not implement Provider interface", configSourceID)
		}

		rootProviders = append(rootProviders, mapProvider)
	}

	rootProviders = append(rootProviders, configmapprovider.NewSimple(rootMap))
	mp.merged = configmapprovider.NewMerge(rootProviders...)

	retrieved, err := mp.merged.Retrieve(ctx, onChange)
	if err != nil {
		return nil, fmt.Errorf("cannot retrive the configuration: %w", err)
	}

	// Get list of value sources.
	valueSources := map[config.ComponentID]configmapprovider.ValueProvider{}
	for configSourceID, configSource := range configSources {
		valueSource, ok := configSource.(configmapprovider.ValueProvider)
		if ok {
			valueSources[configSourceID] = valueSource
		}
	}

	return &valueSourceSubstitutor{onChange: onChange, retrieved: retrieved, valueSources: valueSources}, nil
}

func unmarshalSources(ctx context.Context, rootMap *config.Map, factories component.Factories) (
	configSources map[config.ComponentID]configmapprovider.BaseProvider,
	mergeConfigs []string,
	err error,
) {
	type RootConfig struct {
		ConfigSources []map[string]map[string]interface{} `mapstructure:"config_sources"`
		MergeConfigs  []string                            `mapstructure:"merge_configs"`
	}
	var rootCfg RootConfig
	err = rootMap.Unmarshal(&rootCfg)
	if err != nil {
		return nil, nil, err
	}

	// Unmarshal the "config_sources" section of rootMap and create a BaseProvider for each
	// config source using the factories.ConfigSources.
	configSources = map[config.ComponentID]configmapprovider.BaseProvider{}
	for _, sourceCfg := range rootCfg.ConfigSources {
		for sourceName, settings := range sourceCfg {
			id, err := config.NewComponentIDFromString(sourceName)
			if err != nil {
				return nil, nil, err
			}
			sourceType := id.Type()

			factoryBase, ok := factories.ConfigSources[sourceType]
			if !ok {
				return nil, nil, fmt.Errorf("unknown config source type %q (did you register the config source factory?)", sourceType)
			}

			cfg := factoryBase.CreateDefaultConfig()
			cfg.SetIDName(sourceName)

			// Now that the default config struct is created we can Unmarshal into it,
			// and it will apply user-defined config on top of the default.
			if err := configunmarshaler.Unmarshal(config.NewMapFromStringMap(settings), cfg); err != nil {
				return nil, nil, fmt.Errorf("error reading config of config source %q: %w", sourceType, err)
			}

			factory, ok := factoryBase.(component.ConfigSourceFactory)
			if !ok {
				return nil, nil, fmt.Errorf("config source %q does not implement ConfigSourceFactory", sourceType)
			}

			source, err := factory.CreateConfigSource(ctx, component.ConfigSourceCreateSettings{}, cfg)
			if err != nil {
				return nil, nil, err
			}
			configSources[id] = source
		}
	}

	return configSources, rootCfg.MergeConfigs, nil
}

func (mp *defaultConfigProvider) Shutdown(ctx context.Context) error {
	if mp.merged != nil {
		return mp.merged.Shutdown(ctx)
	}
	return nil
}
