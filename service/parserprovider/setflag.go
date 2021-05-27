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

package parserprovider

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/config/configparser"
)

const setFlagFileType = "properties"

type setFlagProvider struct {
	base ParserProvider
}

// NewSetFlag returns a config.ParserProvider, that wraps a "base" config.ParserProvider, then
// overrides properties from set flag(s) in the loaded Parser.
//
// The implementation reads set flag(s) from the cmd and concatenates them as a "properties" file.
// Then the properties file is read and properties are set to the loaded Parser.
func NewSetFlag(base ParserProvider) ParserProvider {
	return &setFlagProvider{
		base: base,
	}
}

func (sfl *setFlagProvider) Get() (*configparser.Parser, error) {
	flagProperties := getSetFlag()
	if len(flagProperties) == 0 {
		return sfl.base.Get()
	}

	b := &bytes.Buffer{}
	for _, property := range flagProperties {
		property = strings.TrimSpace(property)
		if _, err := fmt.Fprintf(b, "%s\n", property); err != nil {
			return nil, err
		}
	}
	viperFlags := viper.NewWithOptions(viper.KeyDelimiter(configparser.KeyDelimiter))
	viperFlags.SetConfigType(setFlagFileType)
	if err := viperFlags.ReadConfig(b); err != nil {
		return nil, fmt.Errorf("failed to read set flag config: %v", err)
	}

	cp, err := sfl.base.Get()
	if err != nil {
		return nil, err
	}

	// Viper implementation of v.MergeConfig(io.Reader) or v.MergeConfigMap(map[string]interface)
	// does not work properly.  This is b/c if it attempts to merge into a nil object it will fail here
	// https://github.com/spf13/viper/blob/3826be313591f83193f048520482a7b3cf17d506/viper.go#L1709

	// The workaround is to call v.Set(string, interface) on all root properties from the config file
	// this will correctly preserve the original config and set them up for viper to overlay them with
	// the --set params.  It should also be noted that setting the root keys is important.  This is
	// b/c the viper .AllKeys() method does not return empty objects.
	// For instance with the following yaml structure:
	// a:
	//   b:
	//   c: {}
	//
	// viper.AllKeys() would only return a.b, but not a.c.  However otel expects {} to behave
	// the same as nil object in its config file.  Therefore we extract and set the root keys only
	// to catch both a.b and a.c.

	rootKeys := map[string]struct{}{}
	for _, k := range viperFlags.AllKeys() {
		keys := strings.Split(k, configparser.KeyDelimiter)
		if len(keys) > 0 {
			rootKeys[keys[0]] = struct{}{}
		}
	}

	for k := range rootKeys {
		cp.Set(k, cp.Get(k))
	}

	// now that we've copied the config into the viper "overrides" copy the --set flags
	// as well
	for _, k := range viperFlags.AllKeys() {
		cp.Set(k, viperFlags.Get(k))
	}

	return cp, nil
}
