// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package labelfilterprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	require.NoError(t, err)

	factory := NewFactory()
	cfgType := configmodels.Type("labelfilter")
	factories.Processors[cfgType] = factory

	cfgPath := path.Join("testdata", "config.yaml")
	cfg, err := configtest.LoadConfigFile(t, cfgPath, factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	procCfg := cfg.Processors["labelfilter"]
	require.NotNil(t, procCfg)

	flProcCfg := procCfg.(*Config)
	require.NotNil(t, flProcCfg)

	assert.Equal(t, cfgType, flProcCfg.TypeVal)
	assert.Equal(t, "labelfilter", flProcCfg.NameVal)
}
