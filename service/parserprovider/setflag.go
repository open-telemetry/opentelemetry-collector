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

func (sfl *setFlagProvider) Retrieve(ctx context.Context) (Retrieved, error) {
	var ret Retrieved
	var err error
	if ret, err = sfl.base.Retrieve(ctx); err != nil {
		return nil, err
	}

	flagProperties := getSetFlag()
	if len(flagProperties) == 0 {
		return ret, nil
	}

	b := &bytes.Buffer{}
	for _, property := range flagProperties {
		property = strings.TrimSpace(property)
		if _, err = fmt.Fprintf(b, "%s\n", property); err != nil {
			return nil, err
		}
	}

	var props *properties.Properties
	if props, err = properties.Load(b.Bytes(), properties.UTF8); err != nil {
		return nil, err
	}

	// Create a map manually instead of using props.Map() to allow env var expansion
	// as used by original Viper-based configparser.ConfigMap.
	propMap := make(map[string]interface{}, props.Len())
	for _, key := range props.Keys() {
		value, _ := props.Get(key)
		propMap[key] = value
	}
	var confMap *configparser.ConfigMap
	if confMap, err = ret.Get(); err != nil {
		return nil, err
	}
	if err = confMap.MergeStringMap(maps.Unflatten(propMap, ".")); err != nil {
		return nil, err
	}
	return &simpleRetrieved{confMap: confMap}, nil
}

func (sfl *setFlagProvider) Close(context.Context) error {
	return nil
}
