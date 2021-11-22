package defaultconfigprovider

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

func createInlineMapProvider(configFlagValue string, factories component.Factories) (configmapprovider.MapProvider, error) {
	// Parse the command line flag.
	cfgSrcName, selector, paramsConfigMap, err := parseCfgSrcInvocation(configFlagValue)
	if err != nil {
		return nil, fmt.Errorf("invalid format for --config flag value")
	}

	// Find the config source factory.
	factory, ok := factories.ConfigSources[config.Type(cfgSrcName)]
	if !ok {
		var allTypes []string
		for t := range factories.ConfigSources {
			allTypes = append(allTypes, string(t))
		}
		return nil, fmt.Errorf("unknown source type %q (try one of %s)", cfgSrcName, strings.Join(allTypes, ","))
	}

	// Create the config source.
	cfg := factory.CreateDefaultConfig()
	configSource, err := factory.CreateConfigSource(context.Background(), component.ConfigSourceCreateSettings{}, cfg)
	if err != nil {
		return nil, err
	}

	valueProvider, ok := configSource.(configmapprovider.ValueProvider)
	if !ok {
		return nil, fmt.Errorf("config source %s cannot be used from command line because it is not a ValueProvider", cfgSrcName)
	}

	// Convert retrieved value into a config map.
	mapProvider := &mapFromValueProvider{
		valueProvider:    valueProvider,
		selector:         selector,
		paramsConfigMap:  paramsConfigMap,
		configSourceName: cfgSrcName,
	}

	return mapProvider, nil
}
