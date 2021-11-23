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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"bytes"
	"context"
	"strings"

	"github.com/knadh/koanf/maps"
	"github.com/magiconair/properties"

	"go.opentelemetry.io/collector/config"
)

type propertiesMapProvider struct {
	properties []string
}

// NewProperties returns a Provider, that provides a config.Map from the given properties.
//
// Properties must follow the Java properties format, key-value list separated by equal sign with a "."
// as key delimiter.
//  ["processors.batch.timeout=2s", "processors.batch/foo.timeout=3s"]
func NewProperties(properties []string) Provider {
	return &propertiesMapProvider{
		properties: properties,
	}
}

func (pmp *propertiesMapProvider) Retrieve(_ context.Context, onChange func(*ChangeEvent)) (Retrieved, error) {
	if len(pmp.properties) == 0 {
		return &simpleRetrieved{confMap: config.NewMap()}, nil
	}

	b := &bytes.Buffer{}
	for _, property := range pmp.properties {
		property = strings.TrimSpace(property)
		b.WriteString(property)
		b.WriteString("\n")
	}

	var props *properties.Properties
	var err error
	if props, err = properties.Load(b.Bytes(), properties.UTF8); err != nil {
		return nil, err
	}

	// Create a map manually instead of using properties.Map() to allow env var expansion
	// as used by original Viper-based config.Map.
	parsed := make(map[string]interface{}, props.Len())
	for _, key := range props.Keys() {
		value, _ := props.Get(key)
		parsed[key] = value
	}
	prop := maps.Unflatten(parsed, ".")

	return &simpleRetrieved{confMap: config.NewMapFromStringMap(prop)}, nil
}

func (*propertiesMapProvider) Shutdown(context.Context) error {
	return nil
}
