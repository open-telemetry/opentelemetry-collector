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

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configloader"
)

// LoadConfigFile loads a config from file.
func LoadConfigFile(t *testing.T, fileName string, factories component.Factories) (*config.Config, error) {
	// Read yaml config from file
	cp, err := config.NewParserFromFile(fileName)
	require.NoError(t, err)
	// Load the config from viper using the given factories.
	cfg, err := configloader.Load(cp, factories)
	if err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}
