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

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/magiconair/properties"

	"go.opentelemetry.io/collector/config/configparser"
)

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

	var props *properties.Properties
	var err error
	if props, err = properties.Load(b.Bytes(), properties.UTF8); err != nil {
		return nil, err
	}

	// Create a map manually instead of using props.Map() to allow env var expansion
	// as used by original Viper-based configparser.Parser.
	parsed := map[string]interface{}{}
	for _, key := range props.Keys() {
		value, _ := props.Get(key)
		parsed[key] = value
	}

	propertyKoanf := koanf.New(".")
	if err = propertyKoanf.Load(confmap.Provider(parsed, "."), nil); err != nil {
		return nil, fmt.Errorf("failed to read set flag config: %v", err)
	}

	var cp *configparser.Parser
	if cp, err = sfl.base.Get(); err != nil {
		return nil, err
	}

	cp.MergeStringMap(propertyKoanf.Raw())

	return cp, nil
}
