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
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configunmarshaler"

	"go.opentelemetry.io/collector/config"
)

type defaultConfigProvider struct {
	localRoot  configmapprovider.MapProvider
	mergedRoot configmapprovider.MapProvider
	factories  component.Factories
}

// NewDefaultConfigProvider creates a MapProvider that uses the local config file,
// the properties and the additional config sources specified in the config_sources
// and merge_configs sections to build the overall config map.
//
// It first loads config maps from all providers that are specified in the merge_configs
// section, merging them in the order they are specified (i.e. the last in the list has
// the highest precedence), then merges the local config from the file and properties,
// then performs substitution of all config values referenced using $ syntax.
func NewDefaultConfigProvider(configFlagValue string, properties []string, factories component.Factories) (configmapprovider.MapProvider, error) {
	var rootProvider configmapprovider.MapProvider

	if !isValueConfigSourceRef(configFlagValue) {
		// The config flag value is just a config file name.
		fileName := configFlagValue
		rootProvider = configmapprovider.NewFile(fileName)
	} else {
		// The provider is specified directly on the command line. Create it.
		var err error
		rootProvider, err = createInlineMapProvider(configFlagValue, factories)
		if err != nil {
			return nil, err
		}
	}

	// Merge properties.
	rootProvider = configmapprovider.NewMerge(rootProvider, configmapprovider.NewProperties(properties))

	return &defaultConfigProvider{localRoot: rootProvider, factories: factories}, nil
}

func isValueConfigSourceRef(configSourceRef string) bool {
	return strings.Contains(configSourceRef, ":")
}

func (mp *defaultConfigProvider) Retrieve(
	ctx context.Context,
	onChange func(event *configmapprovider.ChangeEvent),
) (configmapprovider.RetrievedMap, error) {
	// Retrieve the local first.
	r, err := mp.localRoot.Retrieve(ctx, onChange)
	if err != nil {
		return nil, err
	}
	localRootMap, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Unmarshal config sources.
	configSources, mergeConfigs, err := unmarshalSources(ctx, localRootMap, mp.factories)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	// Iterate over all sources specified in the merge_configs section and create
	// a provider for each.
	var rootProviders []configmapprovider.MapProvider
	for _, configSourceRef := range mergeConfigs {
		var mapProvider configmapprovider.MapProvider
		mapProvider, err = mp.getConfigSourceMapProvider(configSourceRef, configSources)
		if err != nil {
			return nil, err
		}

		rootProviders = append(rootProviders, mapProvider)
	}

	// Make the local root map the last (highest-precedence) root MapProvider.
	rootProviders = append(rootProviders, configmapprovider.NewSimple(localRootMap))

	// Create a merging provider and retrieve the config from it.
	mp.mergedRoot = configmapprovider.NewMerge(rootProviders...)

	retrieved, err := mp.mergedRoot.Retrieve(ctx, onChange)
	if err != nil {
		return nil, err
	}

	return &valueSubstitutor{
		onChange: onChange, retrieved: retrieved, configSources: configSources,
	}, nil
}

func (mp *defaultConfigProvider) getConfigSourceMapProvider(
	configSourceRef string, configSources map[config.ComponentID]configmapprovider.Provider,
) (configmapprovider.MapProvider, error) {
	if isValueConfigSourceRef(configSourceRef) {
		cfgSrcName, selector, paramsConfigMap, err := parseCfgSrcInvocation(configSourceRef)
		if err != nil {
			return nil, err
		}

		configSource, err := mp.findConfigSource(cfgSrcName, configSources)
		if err != nil {
			return nil, err
		}

		valueProvider, ok := configSource.(configmapprovider.ValueProvider)
		if !ok {
			return nil, fmt.Errorf(
				"config source %q cannot be used in merge_configs section since it does not implement ValueProvider interface",
				configSource,
			)
		}

		mapProvider := &mapFromValueProvider{
			valueProvider:    valueProvider,
			selector:         selector,
			paramsConfigMap:  paramsConfigMap,
			configSourceName: cfgSrcName,
		}
		return mapProvider, nil
	}

	configSource, err := mp.findConfigSource(configSourceRef, configSources)
	if err != nil {
		return nil, err
	}

	mapProvider, ok := configSource.(configmapprovider.MapProvider)
	if !ok {
		return nil, fmt.Errorf(
			"config source %q cannot be used in merge_configs section since it does not implement MapProvider interface",
			configSource,
		)
	}
	return mapProvider, nil

}

func (mp *defaultConfigProvider) findConfigSource(
	cfgSrcName string,
	configSources map[config.ComponentID]configmapprovider.Provider,
) (configmapprovider.Provider, error) {
	configSourceID, err := config.NewComponentIDFromString(cfgSrcName)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid ID in config source specifier %q: %w", cfgSrcName, err,
		)
	}

	configSource, ok := configSources[configSourceID]
	if !ok {
		return nil, fmt.Errorf(
			"config source %q must be defined in config_sources section", cfgSrcName,
		)
	}
	return configSource, nil
}

func unmarshalSources(ctx context.Context, rootMap *config.Map, factories component.Factories) (
	configSources map[config.ComponentID]configmapprovider.Provider,
	mergeConfigs []string,
	err error,
) {
	var rootCfg configunmarshaler.ConfigSourceSettings
	err = rootMap.Unmarshal(&rootCfg)
	if err != nil {
		return nil, nil, err
	}

	// Unmarshal the "config_sources" section of rootMap and create a Provider for each
	// config source using the factories.ConfigSources.
	configSources = map[config.ComponentID]configmapprovider.Provider{}
	for _, sourceCfg := range rootCfg.ConfigSources {
		for sourceName, settings := range sourceCfg {
			id, err := config.NewComponentIDFromString(sourceName)
			if err != nil {
				return nil, nil, err
			}
			sourceType := id.Type()

			// See if we have a factory for this config source type.
			factory, ok := factories.ConfigSources[sourceType]
			if !ok {
				return nil, nil, fmt.Errorf("unknown config source type %q (did you register the config source factory?)", sourceType)
			}

			cfg := factory.CreateDefaultConfig()
			cfg.SetIDName(sourceName)

			// Now that the default config struct is created we can Unmarshal into it,
			// and it will apply user-defined config on top of the default.
			if err = configunmarshaler.Unmarshal(config.NewMapFromStringMap(settings), cfg); err != nil {
				return nil, nil, fmt.Errorf("error reading config of config source %q: %w", sourceType, err)
			}

			if err = cfg.Validate(); err != nil {
				return nil, nil, fmt.Errorf("config of config source %q is invalid: %w", sourceType, err)
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
	if mp.mergedRoot != nil {
		return mp.mergedRoot.Shutdown(ctx)
	}
	return nil
}
