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

package envprovider // import "go.opentelemetry.io/collector/confmap/provider/envprovider"

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
)

const schemeName = "env"

type provider struct{}

// New returns a new confmap.Provider that reads the configuration from the given environment variable.
//
// This Provider supports "env" scheme, and can be called with a selector:
// `env:NAME_OF_ENVIRONMENT_VARIABLE`
func New() confmap.Provider {
	return &provider{}
}

func (emp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return confmap.Retrieved{}, fmt.Errorf("%v uri is not supported by %v provider", uri, schemeName)
	}

	content := os.Getenv(uri[len(schemeName)+1:])
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return confmap.Retrieved{}, fmt.Errorf("unable to parse yaml: %w", err)
	}

	return confmap.NewRetrievedFromMap(confmap.NewFromStringMap(data)), nil
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
