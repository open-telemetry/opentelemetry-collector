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
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/config"
)

const bytesSchemeName = "yaml"

type yamlMapProvider struct{}

// NewYAML returns a new Provider that allows to provide yaml bytes.
//
// This Provider supports "yaml" scheme, and can be called with a "location" that follows:
//   bytes-location   = "yaml:" yaml-bytes
//
// Examples:
// `yaml:processors::batch::timeout: 2s`
// `yaml:processors::batch/foo::timeout: 3s`
func NewYAML() Provider {
	return &yamlMapProvider{}
}

func (s *yamlMapProvider) Retrieve(_ context.Context, location string, _ WatcherFunc) (Retrieved, error) {
	if !strings.HasPrefix(location, bytesSchemeName+":") {
		return Retrieved{}, fmt.Errorf("%v location is not supported by %v provider", location, bytesSchemeName)
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(location[len(bytesSchemeName)+1:]), &data); err != nil {
		return Retrieved{}, fmt.Errorf("unable to parse yaml: %w", err)
	}

	return Retrieved{Map: config.NewMapFromStringMap(data)}, nil
}

func (s *yamlMapProvider) Shutdown(context.Context) error {
	return nil
}
