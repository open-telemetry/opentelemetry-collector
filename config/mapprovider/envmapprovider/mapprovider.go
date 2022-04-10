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

package envmapprovider // import "go.opentelemetry.io/collector/config/mapprovider/envmapprovider"

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/config"
)

const schemeName = "env"

type mapProvider struct{}

// New returns a new config.MapProvider that reads the configuration from the given environment variable.
//
// This Provider supports "env" scheme, and can be called with a selector:
// `env:NAME_OF_ENVIRONMENT_VARIABLE`
func New() config.MapProvider {
	return &mapProvider{}
}

func (emp *mapProvider) Retrieve(_ context.Context, uri string, _ config.WatcherFunc) (config.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return config.Retrieved{}, fmt.Errorf("%v uri is not supported by %v provider", uri, schemeName)
	}

	content := os.Getenv(uri[len(schemeName)+1:])
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return config.Retrieved{}, fmt.Errorf("unable to parse yaml: %w", err)
	}

	return config.Retrieved{Map: config.NewMapFromStringMap(data)}, nil
}

func (*mapProvider) Scheme() string {
	return schemeName
}

func (*mapProvider) Shutdown(context.Context) error {
	return nil
}
