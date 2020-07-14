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

package resourceprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/processor/attraction"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)

	factories.Processors[typeStr] = NewFactory()

	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, cfg.Processors["resource"], &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		AttributesActions: []attraction.ActionKeyValue{
			{Key: "cloud.zone", Value: "zone-1", Action: attraction.UPSERT},
			{Key: "k8s.cluster.name", FromAttribute: "k8s-cluster", Action: attraction.INSERT},
			{Key: "redundant-attribute", Action: attraction.DELETE},
		},
	})

	assert.Equal(t, cfg.Processors["resource/invalid"], &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource/invalid",
		},
	})
}
