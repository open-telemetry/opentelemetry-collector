// Copyright 2019, OpenTelemetry Authors
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

package attributekeyprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

var _ = config.RegisterTestFactories()

func TestLoadConfig(t *testing.T) {

	config, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"))

	require.Nil(t, err)
	require.NotNil(t, config)

	p0 := config.Processors["attribute-key"]
	assert.Equal(t, p0,
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "attribute-key",
				NameVal: "attribute-key",
			},
			KeyReplacements: []KeyReplacement{
				KeyReplacement{
					Key:          "foo",
					NewKey:       "boo",
					Overwrite:    true,
					KeepOriginal: true,
				},
				KeyReplacement{
					Key:          "kie",
					NewKey:       "goo",
					Overwrite:    true,
					KeepOriginal: false,
				},
				KeyReplacement{
					Key:          "ddd",
					NewKey:       "vss",
					Overwrite:    false,
					KeepOriginal: true,
				},
			},
		})
}

func TestLoadConfigEmpty(t *testing.T) {
	factory := processor.GetFactory(typeStr)

	config, err := config.LoadConfigFile(t, path.Join(".", "testdata", "empty.yaml"))

	require.Nil(t, err)
	require.NotNil(t, config)
	p0 := config.Processors["attribute-key"]
	assert.Equal(t, p0, factory.CreateDefaultConfig())
}
