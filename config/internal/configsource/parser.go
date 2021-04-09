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

package configsource

import (
	"context"
	"fmt"

	"github.com/spf13/cast"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configparser"
)

const (
	configSourcesKey = "config_sources"
)

// Private error types to help with testability.
type (
	errInvalidTypeAndNameKey struct{ error }
	errUnknownType           struct{ error }
	errUnmarshalError        struct{ error }
	errDuplicateName         struct{ error }
)

// Load reads the configuration for ConfigSource objects from the given parser and returns a map
// from the full name of config sources to the respective ConfigSettings.
func Load(_ context.Context, v *config.Parser, factories Factories) (map[string]ConfigSettings, error) {

	cfgSrcSettings, err := loadSettings(cast.ToStringMap(v.Get(configSourcesKey)), factories)
	if err != nil {
		return nil, err
	}

	return cfgSrcSettings, nil
}

func loadSettings(css map[string]interface{}, factories Factories) (map[string]ConfigSettings, error) {
	// Prepare resulting map.
	cfgSrcToSettings := make(map[string]ConfigSettings)

	// Iterate over extensions and create a config for each.
	for key, value := range css {
		settingsParser := config.NewParserFromStringMap(cast.ToStringMap(value))

		// TODO: expand env vars.

		// Decode the key into type and fullName components.
		typeStr, fullName, err := configparser.DecodeTypeAndName(key)
		if err != nil {
			return nil, &errInvalidTypeAndNameKey{fmt.Errorf("invalid %s type and name key %q: %v", configSourcesKey, key, err)}
		}

		// Find the factory based on "type" that we read from config source.
		factory := factories[typeStr]
		if factory == nil {
			return nil, &errUnknownType{fmt.Errorf("unknown %s type %q for %s", configSourcesKey, typeStr, fullName)}
		}

		// Create the default config.
		cfgSrcSettings := factory.CreateDefaultConfig()
		cfgSrcSettings.SetName(fullName)

		// Now that the default settings struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := settingsParser.UnmarshalExact(&cfgSrcSettings); err != nil {
			return nil, &errUnmarshalError{fmt.Errorf("error reading %s configuration for %s: %v", configSourcesKey, fullName, err)}
		}

		if cfgSrcToSettings[fullName] != nil {
			return nil, &errDuplicateName{fmt.Errorf("duplicate %s name %s", configSourcesKey, fullName)}
		}

		cfgSrcToSettings[fullName] = cfgSrcSettings
	}

	return cfgSrcToSettings, nil
}
