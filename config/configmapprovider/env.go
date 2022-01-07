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
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/config"
)

const envSchemeName = "env"

type envMapProvider struct{}

// NewEnv returns a new Provider that reads the configuration from the given environment variable.
//
// This Provider supports "env" scheme, and can be called with a selector:
// `env:NAME_OF_ENVIRONMENT_VARIABLE`
func NewEnv() Provider {
	return &envMapProvider{}
}

func (emp *envMapProvider) Retrieve(_ context.Context, location string, _ WatcherFunc) (Retrieved, error) {
	if !strings.HasPrefix(location, envSchemeName+":") {
		return nil, fmt.Errorf("%v location is not supported by %v provider", location, envSchemeName)
	}

	content := os.Getenv(location[len(envSchemeName)+1:])
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("unable to parse yaml: %w", err)
	}

	return NewRetrieved(func(ctx context.Context) (*config.Map, error) {
		return config.NewMapFromStringMap(data), nil
	})
}

func (*envMapProvider) Shutdown(context.Context) error {
	return nil
}
