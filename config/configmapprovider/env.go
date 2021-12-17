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
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/config"
)

type envMapProvider struct {
	envName string
}

// NewEnv returns a new Provider that reads the configuration from the given environment variable.
func NewEnv(envName string) Provider {
	return &envMapProvider{
		envName: envName,
	}
}

func (emp *envMapProvider) Retrieve(_ context.Context, _ func(*ChangeEvent)) (Retrieved, error) {
	if emp.envName == "" {
		return nil, errors.New("config environment not specified")
	}

	content := os.Getenv(emp.envName)

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
