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
	"context"
	"fmt"
	"strings"

	"github.com/knadh/koanf/maps"
	"github.com/magiconair/properties"

	"go.opentelemetry.io/collector/config/configparser"
)

type setFlagProvider struct{}

// NewSetFlag returns a ParserProvider, that provides a configparser.ConfigMap from set flag(s) properties.
//
// The implementation reads set flag(s) from the cmd and concatenates them as a "properties" file.
// Then the properties file is read and properties are set to the loaded Parser.
func NewSetFlag() ParserProvider {
	return &setFlagProvider{}
}

func (sfl *setFlagProvider) Get(context.Context) (*configparser.ConfigMap, error) {
	flagProperties := getSetFlag()
	if len(flagProperties) == 0 {
		return configparser.NewConfigMap(), nil
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
	// as used by original Viper-based configparser.ConfigMap.
	parsed := make(map[string]interface{}, props.Len())
	for _, key := range props.Keys() {
		value, _ := props.Get(key)
		parsed[key] = value
	}
	prop := maps.Unflatten(parsed, ".")

	return configparser.NewConfigMapFromStringMap(prop), nil
}

func (sfl *setFlagProvider) Close(context.Context) error {
	return nil
}
