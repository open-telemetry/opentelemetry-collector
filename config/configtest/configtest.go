// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configtest

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
)

// NewViperFromYamlFile creates a viper instance that reads the given fileName as yaml config
// and can then be used to unmarshal the file contents to objects.
// Example usage for testing can be found in configtest_test.go
func NewViperFromYamlFile(t *testing.T, fileName string) *viper.Viper {
	// Read yaml config from file
	v := config.NewViper()
	v.SetConfigFile(fileName)
	require.NoErrorf(t, v.ReadInConfig(), "unable to read the file %v", fileName)

	return v
}

// LoadConfigFile loads a config from file.
func LoadConfigFile(t *testing.T, fileName string, factories component.Factories) (*configmodels.Config, error) {
	v := NewViperFromYamlFile(t, fileName)

	// Load the config from viper using the given factories.
	cfg, err := config.Load(v, factories)
	if err != nil {
		return nil, err
	}
	return cfg, config.ValidateConfig(cfg, zap.NewNop())
}
